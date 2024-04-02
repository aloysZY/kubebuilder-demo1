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

// groupversion_info.go、zz_generated.deepcopy.go 文件，它们的作用是什么呢? 这与 Scheme 模块的原理有关，即 Scheme 通过这 2 个文件实现了 CRD 的注册及资源的拷

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AppSpec defines the desired state of App
type AppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of App. Edit app_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// AppStatus defines the observed state of App
type AppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// App is the Schema for the apps API
type App struct {
	// metav1.TypeMeta：保存了资源的Group、Version、Kind
	// metav1.ObjectMeta：保存了资源对象的名称和namespace
	// Spec：期望状态，例如deployment在创建时指定了pod有三个副本
	// Status：真实状态，例如deployment在创建后只有一个副本(其他的还没有创建成功)，大多数资源对象都有此字段，不过ConfigMap是个例外（想想也是，配置信息嘛，配成啥就是啥，没有什么期望值和真实值的说法）；
	// 还有一个数据结构，就是Guestbook对应的列表GuestbookList，就是单个资源对象的集合；
	// guestbook_types.go所在目录下还有两个文件：groupversion_info.go定义了Group和Version，以及注册到scheme时用到的实例SchemeBuilder，zz_generated.deepcopy.go用于实现实例的深拷贝，它们都无需修改，了解即可；
	metav1.TypeMeta `json:",inline"`
	// Finalizers []string
	// Finalizers 是在对象删除之前需要执行的逻辑，比如你给资源类型中的每个对象都创建 了对应的外部资源，并且希望在 Kuebernetes 删除对应资源的同时删除关联的外部资源， 那么可以通过 Finalizers 来实现。当 Finalizers 字段存在时，相关资源不允许被强制删除。 所有的对象在被彻底删除之前，它的 Finalizers 字段必须为空，即必须保证在所有对象被 彻底删除之前，与它关联的所有相关资源已被删除
	// 当metadata.DeletionTimestamp字段为非空时，client-go-Controller 监听对象并执行对应 Finalizers 的动作，在所有动作执行完成后，将该 Finalizer 从列表中移除。一旦 Finalizers 列表为空，就意味着所有 Finalizer 都被执行过，最终 Kubernetes 会删除该资源。
	// 在 Operator client-go-Controller 中，最重要的逻辑就是 Reconcile 方法，Finalizers 也是在 Reconcile 中实现的
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	// Register 里面也是调用的AddKnownTypes进行将自己的类型注册到Scheme
	// Register 需要传入的是g v 和k ，g v在 SchemeBuilder 初始化的时候进行了定义
	SchemeBuilder.Register(&App{}, &AppList{})
}
