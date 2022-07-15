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
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CSIDriverEndpoint
// +k8s:openapi-gen=true
type CSIDriverEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CSIDriverEndpointSpec   `json:"spec,omitempty"`
	Status CSIDriverEndpointStatus `json:"status,omitempty"`
}

// CSIDriverEndpointList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CSIDriverEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CSIDriverEndpoint `json:"items"`
}

// CSIDriverEndpointSpec defines the desired state of CSIDriverEndpoint
type CSIDriverEndpointSpec struct {
}

var _ resource.Object = &CSIDriverEndpoint{}
var _ resourcestrategy.Validater = &CSIDriverEndpoint{}

func (in *CSIDriverEndpoint) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *CSIDriverEndpoint) NamespaceScoped() bool {
	return false
}

func (in *CSIDriverEndpoint) New() runtime.Object {
	return &CSIDriverEndpoint{}
}

func (in *CSIDriverEndpoint) NewList() runtime.Object {
	return &CSIDriverEndpointList{}
}

func (in *CSIDriverEndpoint) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "discovery.storage.k8s.io",
		Version:  "v1alpha1",
		Resource: "csidriverendpoints",
	}
}

func (in *CSIDriverEndpoint) IsStorageVersion() bool {
	return true
}

func (in *CSIDriverEndpoint) Validate(ctx context.Context) field.ErrorList {
	// TODO(user): Modify it, adding your API validation here.
	return nil
}

var _ resource.ObjectList = &CSIDriverEndpointList{}

func (in *CSIDriverEndpointList) GetListMeta() *metav1.ListMeta {
	return &in.ListMeta
}

// CSIDriverEndpointStatus defines the observed state of CSIDriverEndpoint
type CSIDriverEndpointStatus struct {
}

func (in CSIDriverEndpointStatus) SubResourceName() string {
	return "status"
}

// CSIDriverEndpoint implements ObjectWithStatusSubResource interface.
var _ resource.ObjectWithStatusSubResource = &CSIDriverEndpoint{}

func (in *CSIDriverEndpoint) GetStatus() resource.StatusSubResource {
	return in.Status
}

// CSIDriverEndpointStatus{} implements StatusSubResource interface.
var _ resource.StatusSubResource = &CSIDriverEndpointStatus{}

func (in CSIDriverEndpointStatus) CopyTo(parent resource.ObjectWithStatusSubResource) {
	parent.(*CSIDriverEndpoint).Status = in
}
