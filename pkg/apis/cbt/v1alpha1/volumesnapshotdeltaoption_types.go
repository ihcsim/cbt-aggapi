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
	"net/url"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource/resourcestrategy"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VolumeSnapshotDeltaOption
// +k8s:openapi-gen=true
type VolumeSnapshotDeltaOption struct {
	metav1.TypeMeta `json:",inline"`

	// Set to true to fetch all the changed block entries.
	FetchCBD bool `json:"fetchCBD"`

	// Define the maximum number of entries to return in the response.
	Limit uint64 `json:"limit"`

	// Offset defines the start of the block index in the response.
	Offset uint64 `json:"offset"`
}

func (v *VolumeSnapshotDeltaOption) ConvertFromUrlValues(values *url.Values) error {
	const (
		defaultLimit  = 256
		defaultOffset = 0
	)

	if value := values.Get("fetchcbd"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			parsed = false
		}
		v.FetchCBD = parsed
	}

	if value := values.Get("limit"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			parsed = defaultLimit
		}
		v.Limit = parsed
	}

	if value := values.Get("offset"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			parsed = defaultOffset
		}
		v.Limit = parsed
	}

	return nil
}

var _ resourcestrategy.Validater = &VolumeSnapshotDeltaOption{}

func (in *VolumeSnapshotDeltaOption) NamespaceScoped() bool {
	return true
}

func (in *VolumeSnapshotDeltaOption) New() runtime.Object {
	return &VolumeSnapshotDeltaOption{}
}

func (in *VolumeSnapshotDeltaOption) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cbt.storage.k8s.io",
		Version:  "v1alpha1",
		Resource: "volumesnapshotdeltaoptions",
	}
}

func (in *VolumeSnapshotDeltaOption) IsStorageVersion() bool {
	return true
}

func (in *VolumeSnapshotDeltaOption) Validate(ctx context.Context) field.ErrorList {
	// TODO(user): Modify it, adding your API validation here.
	return nil
}
