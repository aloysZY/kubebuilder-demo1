/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"kubebuilder-demo1/api/v1beta1"
	aloysv1beta1 "kubebuilder-demo1/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// controller实现业务需求，在operator开发过程中，尽管业务逻辑各不相同，但有两个共性
// Status(真实状态)是个数据结构，其字段是业务定义的，其字段值也是业务代码执行自定义的逻辑算出来的；
// 业务核心的目标，是确保Status与Spec达成一致，例如deployment指定了pod的副本数为3，如果真实的pod没有三个，deployment的controller代码就去创建pod，如果真实的pod超过了三个，deployment的controller代码就去删除pod;

// AppReconciler reconciles a App object
// 操作资源对象时用到的客户端工具client.Client、Kind和数据结构的关系Scheme
type AppReconciler struct {
	// 这里 Client 的作用比较清晰，就是在 CRD client-go-Controller 执行协调的过程中，需要通过 ClientCRUD(Create、Retrieve、Update、Delete)CRD，即 Get、Create 等方法。所以 DemoReconciler 的结构体第一个元素的对象，指向的是 client-go-Controller-runtime 包中的 Client 接口对象，它设计了必要的方法，如 Get、List、Update 等
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aloys.aloys.tech,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aloys.aloys.tech,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aloys.aloys.tech,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the App object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile

// Reconcile controller.go是operator的业务核心，而controller的核心是其Reconcile方法，将来咱们的大部分代码都是写在这里面的，主要做的事情就是获取status，然后让status和spec达成一致
// 关于status，官方的一段描述值得重视，资源对象的状态应该是每次重新计算出来的，这里以deployment为例，想知道当前有多少个pod，有两种方法，第一种准备个字段记录，每次对pod的增加和删除都修改这个字段，于是读这个字段就知道pod数量了，第二种方法是每次用client工具去实时查询pod数量(cache(缓存了资源当前状态))，目前官方明确推荐使用第二种方法：
// Reconciler(协调器)是提供给 client-go-Controller 的一个函数，可以随时使用对象的 Name 和 Namespace 对其进行调用。当它被调用时，Reconciler 将确保集群中资源的状态和 预设的状态保持一致。例如，ReplicaSet 指定 5 个副本，但系统中仅存在 3 个 Pod 时， Reconciler 将再创建 2 个 Pod，并向 Pod 的 OwnerReference 中添加该 ReplicaSet 的名 称，同时设置“controller=true”属性。
// Reconciler 需要开发者自己实现，并在创建 client-go-Controller 时，通过 Builder.Complete() 或 Bu- ilder.Build() 方法传递给 client-go-Controller。Reconciler 接口定义在 pkg/reconcile/reconcile.go 下， 只有一个该方法:Reconcile(context.Context，Request) (Result，error)。
// 该方法中 Request 包含了需要处理对象的 Name 和 Namespace，Result 决定了是否需要将对象重新加入队列以及如何加入队列
//
//	type Request struct {
//	     types.NamespacedName
//	}
//
// type Result struct {
// // Requeue 告诉 client-go-Controller 是否需要重新将对象加入队列，默认为 False Requeue bool
// // RequeueAfter 大于 0 表示 client-go-Controller 需要在设置的时间间隔后，将对象重新加入队列
// // 注意，当设置了RequeueAfter，就表示Requeue为True，即无须RequeueAfter与 Requeue=True 被同时设置
//
//	     RequeueAfter time.Duration
//	}
//
// Reconciler 主要有以下特性。
// (1)包含 client-go-Controller 的所有业务逻辑。
// (2)Reconciler 通常在单个对象类型上工作，某个 Reconciler 一般只会处理一种类型的资源。
// (3)提供了待处理对象的 Name 和 Namespace。
// (4)协调者不关心负责触发协调的事件内容或事件类型。无论是对象的增加、删除还 是更新操作，Reconciler 中接收的都是对象的名称和命名空间。
// https://zhuanlan.zhihu.com/p/628496918
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	app := &v1beta1.App{}
	// 从缓存中获取app
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 声明 Finalizer 字段，由前文可知，类型为字符串
	// 自定义 Finalizer 的标识符包含一个域名、一个正向斜线和 Finalizer 的名称
	myFinalizerName := "storage.finalizers.tutorial.kubebuilder.io"
	// 通过检查 DeletionTimestamp 字段是否为0，判断资源是否被删除
	if app.ObjectMeta.DeletionTimestamp.IsZero() {
		// 如果 DeletionTimestamp 字段为0 ，说明资源未被删除，此时需要检测是否存在 Finalizer，如果不存在，则添加，并更新到资源对象中
		if !containsString(app.ObjectMeta.Finalizers, myFinalizerName) {
			app.ObjectMeta.Finalizers = append(app.ObjectMeta.
				Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), app); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// 如果DeletionTimestamp 字段不为 0 ，说明对象处于删除状态中
		if containsString(app.ObjectMeta.Finalizers, myFinalizerName) {
			// 如果存在 Finalizer 且与上述声明的 finalizer 匹配，那么执行对应的 hook 逻
			if err := r.deleteExternalResources(app); err != nil {
				// 如果删除失败，则直接返回对应的 err，client-go-Controller 会自动执行重试逻辑
				return ctrl.Result{}, err
			}
			// 如果对应的 hook 执行成功，那么清空 finalizers，Kuebernetes删除对应资源
			app.ObjectMeta.Finalizers = removeString(app.ObjectMeta.
				Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), app); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	// ctrl.Result{true} 表示从新入队，不是进行重试，如果不设置，失败是进行重试的
	return ctrl.Result{}, nil
}

// 使用自定义的Event
type filterEvent struct {
}

func (fe filterEvent) Create(event event.CreateEvent) bool {
	return false
}
func (fe filterEvent) Delete(event event.DeleteEvent) bool {
	return false
}
func (fe filterEvent) Update(event event.UpdateEvent) bool {
	return false
}
func (fe filterEvent) Generic(event event.GenericEvent) bool {
	return false
}

// SetupWithManager sets up the controller with the Manager.
// SetupWithManager方法，在main.go中有调用，指定了Guestbook这个资源的变化会被manager监控，从而触发Reconcile方法：
// CRD 的 client-go-Controller 初始化的核心代码是 SetupWithManager 方法，借助这个方法，就可以完成 CRD 在 Manager 对象中的安装，最后通过 Manager 对 象的 start 方法来完成 CRD client-go-Controller 的运行
// SetupWithManager 使用 Manager 设置 client-go-Controller
// 它首先借助 client-go-Controller-runtime 包初始化 Builder 对象，当它完成 Complete 方法时，实 际完成了 CRD Reconciler 对象的初始化，而这个对象是一个接口方法，它必须实现 Reconcile 方法。
// ctrl.NewControllerManagedBy 方法实际借助 client-go-Controller-runtime 完成了 Builder 对象的构建，并借助（Complete）关联 CRD API 定 义的 Scheme 信息，从而得知 CRD 的 client-go-Controller 需要监听的 CRD 类型、版本等信息。这 个方法的最后一步是 Complete 的过程
// Complete具体过程：在构建 client-go-Controller 的方法中最重要的两个步骤是 doController 和 doWatch。在 doController 的过程中，实际的核心步骤是完成 client-go-Controller 对象的构建，从而实现基于 Scheme 和 client-go-Controller 对象的 CRD 的监听流程。而在构建 client-go-Controller 的过程中，它的 do 字段实际对应的是 Reconciler 接口类型定义的方法，也就是在 client-go-Controller 对象生成之后， 必须实现这个定义的方法。它是如何使 Reconciler 对象同 client-go-Controller 产生联系的?实际上， 在 client-go-Controller 初始化的过程中，借助了 Options 参数对象中设计的 Reconciler 对象，并将 其传递给了 client-go-Controller 对象的 do 字段。所以当我们调用 SetupWithManager 方法的时候， 不仅完成了 client-go-Controller 的初始化，还完成了 client-go-Controller 监听资源的注册与发现过程（doWatch），同时 将 CRD 的必要实现方法(Reconcile 方法)进行了再现。至此，我们完成了 client-go-Controller 的 初始化分析，
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// 通过 ControllerManagedBy(m manager.Manager)*Builder 方法实例化一个 Builder 对象，其中传入的 Manager 提供创建 client-go-Controller 所需的依赖。
	return ctrl.NewControllerManagedBy(mgr).
		// For(&aloysv1beta1.App{}, builder.WithPredicates(filterEvent{})).
		For(&aloysv1beta1.App{}).
		// 其中For和Owns是等同与Watches。For的第二个参数默认为EnqueueRequestForObject。Owns的第二个参数默认为EnqueueRequestForOwner
		// ControllerManagedBy(manager).
		//        For(&appsv1.ReplicaSet{}).
		//        Owns(&corev1.Pod{}).
		//        Build(replicaSetReconcile)
		// 使用 For(object client.Object，opts ...ForOption)方法设置需要监听的资源 类型。除了 For() 方法外
		// func (blder *Builder) Owns(object client.Object, opts ...OwnsOption) *Builder{} 监听Object，并将Object对应的Owner加入队列。例如，在上面的例子中监听Pod对象，根据 Pod 的 Owner 将 Pod 所属的 ReplicaSet 资源加入队列例如，使用Predicates 设置事件的过滤器。
		// 设置 client-go-Controller 的其他属性
		// func (blder *Builder) Watches(src source.Source, eventhandler handler. EventHandler, opts ...WatchesOption) *Builder{}  监听指定资源，使用指定方法对事件进行处理。建议使用For()和Owns()，而不是直接使用 Watches() 方法
		// 其中 ForOption、OwnsOption、WatchesOption 主要用于设置监听属性
		// Predicate(过滤器)是 client-go-Controller.Watch 的可选参数（使用watche进行监听资源可以自定义过滤器)，用于过滤事件（哪些事件需要进行处理）。其接口与部分 实现在 pkg/predicate/predicate.go 下。接口见代码清单 3-35，4 个方法分别对应 4 种类 型的事件过滤，如果通过过滤，则返回 True。
		// type Predicate interface {
		//      Create(event.CreateEvent) bool
		//      Delete(event.DeleteEvent) bool
		//      Update(event.UpdateEvent) bool
		//      Generic(event.GenericEvent) bool
		// }
		// client-go-Controller-runtime 内置了 5 种 Predicate 的实现
		// (1)Funcs:是一个基本结构，结构包含 4 个函数对象成员，分别是 Predicate 的 4 个 方法的实现。开发者需要根据自己的需求设置相应的成员 ，对于未设置的成员，默认会接 受所有对应的事件
		// type Funcs struct {
		//   CreateFunc func(event.CreateEvent) bool
		//   DeleteFunc func(event.DeleteEvent) bool
		//   UpdateFunc func(event.UpdateEvent) bool
		//   GenericFunc func(event.GenericEvent) bool
		// }
		// (2)ResourceVersionChangedPredicate:只实现了 Update 事件过滤的方法，过滤掉 资源对象 ResourceVersion 未改变的 Update 事件，其他如 Create、Delete 类型的事件直 接接受。
		// (3)GenerationChangedPredicate:类 似 于 ResourceVersionChangedPredicate， 也 只实现了 Update 事件的过滤。
		// GenerationChangedPredicate 会跳过资源对象 metadata.generation 未改变的事件。 当对对象的 Spec 字段进行写操作时，Kubernetes API 服务器会累加对象的 metadata. generation 字段。因此，GenerationChangedPredicate 允许 client-go-Controller 忽略 Spec 未更改 而仅元数据 Metadata 或状态字段 Status 发生更改的更新事件。需要注意的是，对于开发 者定义的 CRD，只有当开启了状态子资源时，metadata.generation 字段才会增加。
		// 上面提到的仅在写入 Spec 字段时 metadata.generation 字段才增加的情况，并不 适用于所有的 API 对象，例如 Deployment 对象，在写入 metadata.annotationss 时， metadata.generation 也会增加。另外，由于使用了此 Predicate，client-go-Controller 的同步不会 被只包含状态(Status)更改的事件触发，因此，无法用于同步或恢复对象的状态值。
		// (4)AnnotationChangedPredicate:只实现了 Update 事件的过滤，此 Predicate 跳过 对象的 Annotations 字段无变化的更新事件，可以与 GenerationChangedPredicate 一起使 用，用于同时需要响应对象 Spec 与 Annotation 字段更新的 client-go-Controller
		// 5)LabelChangedPredicate:只实现了 Update 事件的过滤，跳过标签(Label)未改 变的 Update 时间，也可以和上面 AnnotationChangedPredicate 一样，结合 Generation- Changed Predicate 用于同时响应对象 Spec 与 Label 字段更新的 client-go-Controller。
		// 除以上 5 种 Predicate 外，还有两个代表逻辑运算符的方法——Or() 与 And()，两个方 法可以传递多个 Predicate 接口，最终返回一个 Predicate，代表逻辑运算结果。
		// 从上面的例子可以看到，Predicate 可以通过 client-go-Controller 的 Watch() 方法设置。另 外，也可以在创建 client-go-Controller 时，将 Predicate 通过 Builder.WithEventFilter() 传递到 client-go-Controller 中，或是通过 pkg/builder/options.go 下的 WithPredicates() 方法，转换成 实现 ForOption、OwnsOption、WatchesOption 接口的 builder.Predicates 结构，在 Buil- der.For()、Builder.Owns()、Builder.Watches() 方法中设置。
		// Predicate 主要有以下特性。
		// (1)接受一个事件，并将该事件是否通过过滤条件的结果返回。如果通过，该事件将 被加入待处理事件队列中。
		// (2)Predicate 是可选项，可以不设置。如果不设置，默认事件都将被加入待处理事件 队列中。
		// (3)用户可以使用内置的 Predicate，但是可以设置自定义 Predicates
		//  For(&webappv1beta1.Guestbook{}, builder.WithPredicates(filterEvent{})).

		// EventHandler(事件句柄)是 client-go-Controller.Watch 的参数，当事件产生时，EventHandler 将返回对象的 Name 和 Namespace，作为 Request 被添加到待处理事件队列中。例如， 将来自 Source 的 Pod Create 事件提供给 EnqueueHandler，EventHandler 将生成一个 Request 添加到队列中，这个 Request 包含该 Pod 的 Name 和 Namespace。
		// EventHandler 主要有以下特性。
		// (1)通过将 Request 加入队列处理一个或多个对象的事件。 (2)可以将一个事件映射为另一个相同类型对象的 Request。
		// (3)可以将一个事件映射为另一个不同类型对象的 Request。例如，将一个 Pod 事件 映射为一个纳管这个 Pod 的 ReplicaSet 类型 Request。
		// (4)用户应该只使用提供的 Eventhandler 实现，而不是使用自定义 Eventhandler 实现。
		// EventHandler就是当事件产生的时候，可以添加多个 Name 和 Namespace到队列进行处理
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		// WithOptions(controller.Options{CacheSyncTimeout: time.Microsecond * 100}).
		// WithOptions(controller.Options{})

		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func (r *AppReconciler) deleteExternalResources(app *v1beta1.App) error {
	// 删除 app关联的外部资源逻 需要确保实现是幂等的
	return nil
}
