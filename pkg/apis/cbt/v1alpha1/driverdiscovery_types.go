/*
Copyright 2022.

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

package v1alpha1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource/resourcestrategy"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=driverdiscoveries,strategy=DriverDiscoveryStrategy,shortname=dds

// DriverDiscovery
// +k8s:openapi-gen=true
type DriverDiscovery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DriverDiscoverySpec `json:"spec,omitempty"`
}

// DriverDiscoveryList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DriverDiscoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DriverDiscovery `json:"items"`
}

// DriverDiscoverySpec defines the desired state of DriverDiscovery
type DriverDiscoverySpec struct {
	Driver      string `json:"driverName"`
	CBTEndpoint string `json:"cbtEndpoint"`
	CABundle    string `json:"caBundle"`
}

var _ resource.Object = &DriverDiscovery{}
var _ resourcestrategy.Validater = &DriverDiscovery{}

func (in *DriverDiscovery) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *DriverDiscovery) NamespaceScoped() bool {
	return false
}

func (in *DriverDiscovery) New() runtime.Object {
	return &DriverDiscovery{}
}

func (in *DriverDiscovery) NewList() runtime.Object {
	return &DriverDiscoveryList{}
}

func (in *DriverDiscovery) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cbt.storage.k8s.io",
		Version:  "v1alpha1",
		Resource: "driverdiscoveries",
	}
}

func (in *DriverDiscovery) IsStorageVersion() bool {
	return true
}

func (in *DriverDiscovery) Validate(ctx context.Context) field.ErrorList {
	// TODO(user): Modify it, adding your API validation here.
	return nil
}

var _ resource.ObjectList = &DriverDiscoveryList{}

func (in *DriverDiscoveryList) GetListMeta() *metav1.ListMeta {
	return &in.ListMeta
}
