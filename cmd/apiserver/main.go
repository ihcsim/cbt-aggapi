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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	storagebackend "k8s.io/apiserver/pkg/storage/storagebackend"
	storagebackendfactory "k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/apiserver-runtime/pkg/builder"

	// +kubebuilder:scaffold:resource-imports
	"github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	cbtv1alpha1 "github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	cbtclient "github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	cbtstorage "github.com/ihcsim/cbt-aggapi/pkg/storage"
)

func main() {
	clientset, err := cbtClientset()
	if err != nil {
		klog.Fatal(err)
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.VolumeSnapshotDelta{})
	codec := serializer.NewCodecFactory(scheme).LegacyCodec(v1alpha1.SchemeGroupVersion)
	storageConfig := storagebackend.NewDefaultConfig(
		v1alpha1.SchemeGroupVersion.Group, codec)
	storageConfig.Transport = storagebackend.TransportConfig{
		ServerList: []string{"http://etcd-svc:2379"},
	}
	storageConfigResource := storageConfig.ForResource(v1alpha1.SchemeGroupResource)

	etcdStorage, _, err := storagebackendfactory.Create(
		*storageConfigResource,
		(&cbtv1alpha1.VolumeSnapshotDelta{}).New)
	if err != nil {
		klog.Fatal(err)
	}

	apiserver := builder.APIServer.
		// +kubebuilder:scaffold:resource-register
		WithResource(&cbtv1alpha1.DriverDiscovery{}).
		WithResourceAndHandler(&cbtv1alpha1.VolumeSnapshotDelta{},
			cbtstorage.NewCustomStorage(
				&cbtv1alpha1.VolumeSnapshotDelta{},
				clientset,
				etcdStorage,
			)).
		WithLocalDebugExtension()

	if err := apiserver.Execute(); err != nil {
		klog.Fatal(err)
	}
}

func cbtClientset() (cbtclient.Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return cbtclient.NewForConfig(restConfig)
}
