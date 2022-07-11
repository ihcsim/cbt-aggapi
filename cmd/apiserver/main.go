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

package main

import (
	flag "github.com/spf13/pflag"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog"
	"sigs.k8s.io/apiserver-runtime/pkg/builder"

	// +kubebuilder:scaffold:resource-imports
	cbtv1alpha1 "github.com/ihcsim/cbt-controller/pkg/apis/cbt/v1alpha1"
	grpccbt "github.com/ihcsim/cbt-controller/pkg/grpc"
	"github.com/ihcsim/cbt-controller/pkg/storage"
)

var grpcTarget = flag.String("target", ":9779", "Address of the GRPC server")

func main() {
	grpcOpts := grpc.WithTransportCredentials(insecure.NewCredentials())
	clientConn, err := grpc.Dial(*grpcTarget, grpcOpts)
	if err != nil {
		klog.Fatal(err)
	}
	defer func() {
		if err := clientConn.Close(); err != nil {
			klog.Error(err)
		}
	}()

	// @TODO
	// - authn/authz
	// - remove CBD from status subresource

	grpcClient := grpccbt.NewVolumeSnapshotDeltaServiceClient(clientConn)
	if err := builder.APIServer.
		// +kubebuilder:scaffold:resource-register
		WithResourceAndHandler(&cbtv1alpha1.VolumeSnapshotDelta{},
			storage.NewStorageProvider(
				&cbtv1alpha1.VolumeSnapshotDelta{},
				grpcClient)).
		WithLocalDebugExtension().
		WithFlagFns(addCustomFlags).
		Execute(); err != nil {
		klog.Fatal(err)
	}
}

func addCustomFlags(set *flag.FlagSet) *flag.FlagSet {
	set.AddFlag(flag.Lookup("target"))
	return set
}
