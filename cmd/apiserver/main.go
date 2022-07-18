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
	"fmt"
	"time"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sigs.k8s.io/apiserver-runtime/pkg/builder"

	// +kubebuilder:scaffold:resource-imports
	cbtv1alpha1 "github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	cbtclient "github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	genericinformers "github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/informers/externalversions"
	cbtinformers "github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/informers/externalversions/cbt"
	cbtstorage "github.com/ihcsim/cbt-aggapi/pkg/storage/cbt"
)

func main() {
	informers, informersFactory, err := cbtInformers()
	if err != nil {
		klog.Fatal(err)
	}

	apiserver := builder.APIServer.
		// +kubebuilder:scaffold:resource-register
		WithResource(&cbtv1alpha1.DriverDiscovery{}).
		WithResourceAndHandler(&cbtv1alpha1.VolumeSnapshotDelta{},
			cbtstorage.NewStorageProvider(
				&cbtv1alpha1.VolumeSnapshotDelta{},
				informers,
			)).
		WithPostStartHook("start-informers", func(ctx genericapiserver.PostStartHookContext) error {
			informers.V1alpha1().DriverDiscoveries().Informer().AddEventHandler(
				cache.ResourceEventHandlerFuncs{
					AddFunc:    func(new interface{}) {},
					UpdateFunc: func(old, new interface{}) {},
					DeleteFunc: func(obj interface{}) {},
				},
			)

			informersFactory.Start(ctx.StopCh)
			outcome := informersFactory.WaitForCacheSync(ctx.StopCh)
			for kind, ok := range outcome {
				if !ok {
					return fmt.Errorf("informer cache sync failed. kind: %v", kind)
				}
			}

			return nil
		}).
		WithLocalDebugExtension()

	if err := apiserver.Execute(); err != nil {
		klog.Fatal(err)
	}
}

func cbtInformers() (cbtinformers.Interface, genericinformers.SharedInformerFactory, error) {
	resyncDuration := time.Minute * 10
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}

	client, err := cbtclient.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	informersFactory := genericinformers.NewSharedInformerFactory(client, resyncDuration)
	return informersFactory.Cbt(), informersFactory, nil
}
