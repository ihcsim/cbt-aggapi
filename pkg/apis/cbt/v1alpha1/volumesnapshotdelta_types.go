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

// VolumeSnapshotDelta
// +k8s:openapi-gen=true
type VolumeSnapshotDelta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeSnapshotDeltaSpec   `json:"spec,omitempty"`
	Status VolumeSnapshotDeltaStatus `json:"status,omitempty"`
}

// VolumeSnapshotDeltaList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VolumeSnapshotDeltaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VolumeSnapshotDelta `json:"items"`
}

// VolumeSnapshotDeltaSpec defines the desired state of VolumeSnapshotDelta
type VolumeSnapshotDeltaSpec struct {
	// The name of the base CSI volume snapshot to use for comparison.
	// If not specified, return all changed blocks.
	// +optional
	BaseVolumeSnapshotName string `json:"baseVolumeSnapshotName,omitempty"`

	// The name of the target CSI volume snapshot to use for comparison.
	// Required.
	TargetVolumeSnapshotName string `json:"targetVolumeSnapshotName"`

	// Defines the type of volume. Default to "block".
	// Required.
	Mode string `json:"mode,omitempty"`
}

var _ resource.Object = &VolumeSnapshotDelta{}
var _ resourcestrategy.Validater = &VolumeSnapshotDelta{}

func (in *VolumeSnapshotDelta) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *VolumeSnapshotDelta) NamespaceScoped() bool {
	return true
}

func (in *VolumeSnapshotDelta) New() runtime.Object {
	return &VolumeSnapshotDelta{}
}

func (in *VolumeSnapshotDelta) NewList() runtime.Object {
	return &VolumeSnapshotDeltaList{}
}

func (in *VolumeSnapshotDelta) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cbt.storage.k8s.io",
		Version:  "v1alpha1",
		Resource: "volumesnapshotdeltas",
	}
}

func (in *VolumeSnapshotDelta) IsStorageVersion() bool {
	return true
}

func (in *VolumeSnapshotDelta) Validate(ctx context.Context) field.ErrorList {
	// TODO(user): Modify it, adding your API validation here.
	return nil
}

var _ resource.ObjectList = &VolumeSnapshotDeltaList{}

func (in *VolumeSnapshotDeltaList) GetListMeta() *metav1.ListMeta {
	return &in.ListMeta
}

// VolumeSnapshotDeltaStatus defines the observed state of VolumeSnapshotDelta
type VolumeSnapshotDeltaStatus struct {
	// Captures any error encountered.
	Error string `json:"error,omitempty"`

	// The list of changed block data.
	ChangedBlockDeltas []*ChangedBlockDelta `json:"changedBlockDeltas,omitempty"`
}

type ChangedBlockDelta struct {
	// The block logical offset on the volume.
	Offset uint64 `json:"offset"`

	// The size of the block in bytes.
	BlockSizeBytes uint64 `json:"blockSizeBytes"`

	// The token and other information needed to retrieve the actual data block
	// at the given offset.
	DataToken DataToken `json:"dataToken"`
}

type DataToken struct {
	// The token to use to retrieve the actual data block at the given offset.
	Token string `json:"token"`

	// Timestamp when the token is issued.
	IssuanceTime metav1.Time `json:"issuanceTime"`

	// The TTL of the token in seconds. The expiry time is calculated by adding
	// the time of issuance with this value.
	TTL metav1.Duration `json:"ttl"`
}

func (in VolumeSnapshotDeltaStatus) SubResourceName() string {
	return "status"
}

// VolumeSnapshotDelta implements ObjectWithStatusSubResource interface.
var _ resource.ObjectWithStatusSubResource = &VolumeSnapshotDelta{}

func (in *VolumeSnapshotDelta) GetStatus() resource.StatusSubResource {
	return in.Status
}

// VolumeSnapshotDeltaStatus{} implements StatusSubResource interface.
var _ resource.StatusSubResource = &VolumeSnapshotDeltaStatus{}

func (in VolumeSnapshotDeltaStatus) CopyTo(parent resource.ObjectWithStatusSubResource) {
	parent.(*VolumeSnapshotDelta).Status = in
}
