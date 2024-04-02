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

package main

import (
	"crypto/tls"
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	aloysv1beta1 "kubebuilder-demo1/api/v1beta1"
	"kubebuilder-demo1/internal/controller"
	// +kubebuilder:scaffold:imports
)

var (
	// 初始化一个scheme
	// 当我们操作资源和 apiserver 进行通信的时候，需要根据资源对象类型的 Group、Version、Kind 以及规范定义、编解码等内容构成 Scheme 类型，然后 Clientset 对象就可以来访问和操作这些资源类型了
	// 两个全局变量setupLog用于输出日志无需多说，scheme也是常用的工具，它提供了Kind和Go代码中的数据结构的映射，
	scheme = runtime.NewScheme()
	// 日志配置
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// 将k8s自定义的类型(gvk)进行注入，后续client就可以直接进行操作
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// 注册自己的gvk 到scheme
	utilruntime.Must(aloysv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		// 设置为开发配置警告时使用stacktraces，不采样)，否则将使用Zap生产配置(错误时使用stacktraces，采样)。
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// UseFlagOptions将日志记录器配置为使用通过从CLI解析zap选项标志设置的Options。opts := zap.Options{}
	// ctrl.SetLogger 将zap日志设置为controller-runtime的日志
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancelation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	// ctrl.GetConfigOrDie() 获取集群配置文件句柄信息
	// Manager 初始化是借助于 ctrl.NewManager 方法实现的，而这个方法的位置指向 client-go-Controller-runtime 包的 manager.New 方法，在 New 的方法中，实际是根据传入的参数 进行 Manager 对象的 Scheme、Cache、Client 等模块的初始化构建 ctrl.GetConfigOrDie() 集群配置文件句柄，可以创建 client ， Scheme已经初始化，自建的类型和默认类型都已经初始化
	// Manager 的 New 方法中的 Scheme 变量，是借助 Kubebuilder 工具，根据用户 CRD 生成主程序 main.go 的入口函数 main 方法传入的，即这里的 Scheme 已经绑定了用户 CRD。然后，初始化 Manager 对象 构建出来后，通过 Manager 的 Cache 监听 CRD，一旦 CRD 在集群中创建了，Cache 监听 到发生了变化，就会触发 client-go-Controller 的协调程序 Reconcile 工作
	// ctrl.NewManager 用于创建 Manager，在创建 Manager 的过程中会初始化相应的 Client
	// 如果用户没有指定自己用的Client，那么在setOptionsDefaults函数中会创建 默认的Client
	// 创建Cache用于Client读操作 cache, err := options.NewCache(config, cacheOpts)   cluster.New中初始化
	// 初始化用于写操作的Client writeObj, err := options.ClientBuilder  cluster.New中初始化
	// cluster.Cluster:接 口 类 型，Manager 的 匿 名 成 员，Manager 继 承 了 cluster. Cluster 的所有方法。cluster.Cluster 提供了一系列方法，以获取与集群相关的对象。
	// 开发者可以通过以下几种方式访问 Kubernete 集群中的资源。
	// 1 通过 Manager.GetClient() 可以获取 client.Client，从而对 Kubernetes 资源进行 读写，这也是推荐的方式。在读操作上，client.Client 直接查询 Cache 中的资源，Cache 基于 List-Watch 机制缓存了 Kube-APIServer 中的部分资源。在写操作上，client. Client 会向 Kube-APIServer 发送请求。
	// 2 通过Manager.GetAPIReader()获取client.Reader，client.Reader只用于查询 Kubernetes 资源，但不再使用 Cache，而是直接向 Kube-APIServer 发送请求，效率 相对 client.Client 较低。
	// 另外，还可以通过 cluster.Cluster 获取集群的常用数据。
	// 1 通过 Manager.GetConfig() 获取 Kube-APIServer 的 rest.Config 配置，可用于 k8s. io/client-go 中 ClientSet 的创建。
	// 2 通过 Manager.GetScheme() 获取 Kubernetes 集群资源的 Scheme，可以用于注册 CRD。
	// 3 通过Manager.GetEventRecorderFor()获取EventRecorder，可以用于创建 Kubernetes event 到集群中。
	// 4 通过 Manager.GetRESTMapper() 获取 RESTMapper，存储了 Kube-APIServer 中资源 Resource 与 Kind 的对应关系，可以将 GroupVersionResource 转换为对应的 GroupVersionKind。
	// 5 通过 Manager.GetCache() 获取 Cache。
	// cluster.Cluster 还提供了 SetFields() 接口，用于“注入”client-go-Controller 的依赖。此接口 在创建 client-go-Controller 时作为函数对象保存在 client-go-Controller 中，在 client-go-Controller 启动前调用。

	// ControllerManager 对象是 Manager 的实现，其结构成员中部分成员是直接通过 manager.Options 传入的，或是通过 manager.Options 传入的方法创建的，另外一些 主要的成员如下。
	// (1)cluster.Cluster 接口类型成员 Cluster:提供 Manager 接口中的匿名成员 cluster. Cluster 的实现。
	// (2)[]Runnable 类型成员 leaderElectionRunnables 与 nonLeaderElectionRunnables: 两者存储了所有注册到 Manager 中的 client-go-Controller、WebHook Server 以及自定义对象，按 照是否需要遵循选举机制 []Runnable 类型成员分为两类，nonLeaderElectionRunnables 中 的 Runnable，在调用 Manager.Start() 方法后会立即启动。
	// (3)metricsListener 和 healthProbeListener:类型都为 net.Listener，前者是 Prome- theus 监控服务的监听对象，后者是健康检查服务的监听对象。
	// (4)WebHookServer:WebHook 的服务对象，在 Manager.GetWebhookServer() 方 法被调用时，进行创建并返回。
	// (5)startCache:是函数对象，类型为 func(ctx context.Context)error，用于启动 缓 存 的 同 步。 在 启 动 leaderElectionRunnables 和 nonLeaderElectionRunnables 之 前， Manager 会先调用此方法启动缓存同步，并等待同步完成，启动 Runnable。在实现上， startCache 实际上是 cluster.Cluster.Start() 方法。

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		// (1) Scheme 结构。一般先通过 k8s.io/apimachinery/pkg/runtime 中的 NewScheme() 方法获取 Kubernetes 的 Scheme，然后再将 CRD 注册到 Scheme 中
		// (2)MapperProvider 是一个函数对象，其定义为 func(c *rest.Config)(meta.REST- Mapper，error)，用于定义 Manager 如何获取 RESTMapper。默认通过 k8s.io/client-go 中的 DiscoveryClient 请求获取 Kube-APIServer。
		// (3)Logger 用于定义 Manager 的日志输出对象，默认使用 pkg/internal/log 包下的 全局参数 RuntimeLog。
		// (4)SyncPeriod 参数用于指定 Informer 重新同步并处理资源的时间间隔，默认为 10 小时。此参数也决定了 client-go-Controller 重新同步的时间间隔，每个 client-go-Controller 的时间间隔以此 参数为基准有 10% 的抖动，以避免多个 client-go-Controller 同时进行重新同步。
		// (5)LeaderElection、LeaderElectionResourceLock、LeaderElectionNamespace、 LeaderElectionID 等用于开启和配置 Manager 的选举。其中，LeaderElectionResource- Lock 配置选举锁的类型可以为 Leases、Configmapsleases、Endpointsleases 等，默认 为 Configmapslease。LeaderElectionNamespace、LeaderElectionID 用于配置锁资源的 Namespace 和 Name
		// (6)Namespace 参数用于限制 Manager.Cache 只监听指定 Namespace 的资源，默认 情况下无限制。
		// (7)NewCache 参数的类型为 cache.NewCacheFunc，Manager 会调用此参数创建 Cache，因此，可以用于自定义 Manager 使用的 Cache。在默认情况下，Manager 使用 InformersMap 对象实现 Cache 接口，InformersMap 接口的实现在 pkg/cache/internal/ deleg_map.go 中
		// (8)ClusterBuilder 参 数 的 类 型 为 ClientBuilder 接 口，Manager 会 调 用 此 接 口 创 建 Client， 即 Manager.GetClient() 返 回 的 Client。 在 默 认 情 况 下，Manager 使 用 pkg/ cluster 下的 newClientBuilder 对象创建 Client。
		// (9)ClientDisableCacheFor 参数用于配置 Client，指定某些资源对象的操作不使用缓 存，而是直接操作 Kube-APIServer。
		// (10)EventBroadcaster 参数用于提供 Manager，以获取 EventRecorder，当前已不 推荐使用，因为当 Manager 或 client-go-Controller 的生命周期短于 EventBroadcaster 的生命周期时， 可能会导致 goroutine 泄露。
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer: webhookServer,
		// 监控指标相关的，以及管理controller和webhook的manager，它会一直运行下去直到被外部终止，关于这个manage还有一处要注意的地方，就是它的参数，如果您想让operator在指定namespace范围内生效，还可以在下午的地方新增Namespace参数，如果要指定多个nanespace，就使用cache.MultiNamespacedCacheBuilder(namespaces)参数
		HealthProbeBindAddress: probeAddr,
		//  LeaderElectionConfig *rest.Config 访问选举锁所在的APIServer的配置
		// LeaderElectionReleaseOnCancel bool 当设置为True时，Manager结束前会主动释放选举锁，否则需要等到选举任期结束 // 才能进行新的选举。此选项可以加快重新选举的速度
		// LeaseDuration *time.Duration 候选人强制获取Leader的时间，相当于一个选举任期，默认为15s
		// RenewDeadline *time.Duration当选Leader后，刷新选举信息的间隔，其需要小于LeaseDuration，默认为10s
		// RetryPeriod *time.Duration 候选人尝试竞选的时间间隔默认为2s
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "ada94d48.aloys.tech",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.AppReconciler{
		// 将 Manager 的 Client 传给 client-go-Controller，
		// ClusterBuilder 参 数 的 类 型 为 ClientBuilder 接 口，Manager 会 调 用 此 接 口 创 建 Client， 即 Manager.GetClient() 返 回 的 Client。 在 默 认 情 况 下，Manager 使 用 pkg/ cluster 下的 newClientBuilder 对象创建 Client。
		// 这个方法的实质过程是通过 k8s 的 kubeconfig 文件生成可访问的 restClient 对象，因此，它具备了对 k8s 所有资源的操作方法， 即 CRUD 的过程。
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		// 并且调用 SetupWithManager 方法传入 Manager 进行 client-go-Controller 的初始化
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "App")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	// mgr启动函数
	// MGR 的类型是一个 Interface，底层实际上调用的是 controllerManager 的 Start 方法。 Start 方法的主要逻辑就是启动 Cache、client-go-Controller，将整个事件流运转起来。
	// go cm.startNonLeaderElectionRunnables()根据是否需要选举来选择启动方式 选举和非选举方式的启动逻辑类似，都是先初始化 Cache，再启动 client-go-Controller
	// 启动Cache   cm.waitForCache
	// 启动Controller for _, c := range cm.nonLeaderElectionRunnables
	// Cache 的核心逻辑是初始化内部所有的 Informer，初始化 Informer 后就创建了 Reflector 和内部 client-go-Controller，Reflector 和 client-go-Controller 两个组件是一个“生产者—消费者” 模型，Reflector 负责监听 APIServer 上指定的 GVK 资源的变化，然后将变更写入 delta 队列中，client-go-Controller 负责消费这些变更的事件，然后更新本地 Indexer，最后计算出是创建、 更新，还是删除事件，推给我们之前注册的 Watch Handler（client-go中的事件处理方法 ）
	// (2)Manager.Start() 方法会启动所有注册到 Manager 中的 client-go-Controller。当开启了 Manager 的选举功能时，Manager 会在启动前尝试获取 Leader，只有当选 Leader 成功， Manager 才会启动注册的 client-go-Controller。
	// 除了 client-go-Controller 外，开发者可以通过 Manager.Add(Runnable)方法注册自定义的对象， 例如，注册一个 HTTP Server，只需要自定义的对象实现 Runnable 接口的 Start(context. Context)error() 方法即可。一般在通过 pkg/builder 下的 Builder 创建 client-go-Controller 对象时， Builder 会自动调用 Manager.Add(Runnable)方法将 client-go-Controller 对象注册到 Manager 中。
	// 与 client-go-Controller 相同，在调用 Manager.Start() 后，Manager 会调用自定义对象的 Start(context.Context)error() 方法，用来启动自定义对象。当自定义对象同时实现了 LeaderElectionRunnable 接口的 NeedLeaderElection() 方法时，Manager 会在启动 前判断此自定义对象是否需要遵循选举机制来启动，在默认情况下，对于未实现此接口 的自定义对象，其效果与实现了此接口且返回为 True 时一样。
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
