package cbt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	cbtclient "github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	restregistry "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog"
	builderresource "sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

var _ rest.Connecter = &cbt{}

// NewStorageProvider creates a new instance of a custom storage provider used
// to handle changed block entries.
func NewStorageProvider(
	obj builderresource.Object,
	clientset cbtclient.Interface,
) builderrest.ResourceHandlerProvider {
	return func(s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (restregistry.Storage, error) {
		return &cbt{
			clientset:       clientset,
			namespaceScoped: obj.NamespaceScoped(),
			newFunc:         obj.New,
			newListFunc:     obj.NewList,
			TableConvertor: restregistry.NewDefaultTableConvertor(schema.GroupResource{
				Group:    obj.GetGroupVersionResource().Group,
				Resource: obj.GetGroupVersionResource().Resource,
			}),
		}, nil
	}
}

type cbt struct {
	clientset       cbtclient.Interface
	namespaceScoped bool
	newFunc         func() runtime.Object
	newListFunc     func() runtime.Object
	watch           chan watch.Event
	restregistry.TableConvertor
}

func (c *cbt) New() runtime.Object {
	return c.newFunc()
}

// NewList returns an empty object that can be used with the List call.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (c *cbt) NewList() runtime.Object {
	return c.newListFunc()
}

// NamespaceScoped returns true if the storage is namespaced
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Scoper
func (c *cbt) NamespaceScoped() bool {
	return c.namespaceScoped
}

// Connect returns an http.Handler that will handle the request/response for a given API invocation.
// The provided responder may be used for common API responses. The responder will write both status
// code and body, so the ServeHTTP method should exit after invoking the responder. The Handler will
// be used for a single API request and then discarded. The Responder is guaranteed to write to the
// same http.ResponseWriter passed to ServeHTTP.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Connecter
func (c *cbt) Connect(ctx context.Context, id string, options runtime.Object, r restregistry.Responder) (http.Handler, error) {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		result := &v1alpha1.VolumeSnapshotDelta{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "cbt.storage.k8s.io/v1alpha1",
				Kind:       "VolumeSnapshotDelta",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delta",
				Namespace: "default",
			},
			Spec: v1alpha1.VolumeSnapshotDeltaSpec{
				BaseVolumeSnapshotName:   "base",
				TargetVolumeSnapshotName: "target",
				Mode:                     "block",
			},
			Status: v1alpha1.VolumeSnapshotDeltaStatus{
				Error: "",
			},
		}

		opts, ok := options.(*v1alpha1.VolumeSnapshotDeltaOption)
		if !ok {
			http.Error(resp, "failed to read VolumeSnapshotDelta options", http.StatusInternalServerError)
			return
		}

		if !opts.FetchCBD {
			writeResponse(resp, result)
			return
		}

		// find the CSI driver
		obj, err := c.clientset.CbtV1alpha1().DriverDiscoveries().Get(ctx, "example.csi.k8s.io", metav1.GetOptions{})
		if err != nil {
			http.Error(resp, fmt.Sprintf("failed to discover CSI driver: %s", err), http.StatusInternalServerError)
			return
		}
		klog.Infof("discovered CSI driver: %s", obj.GetName())

		// send request to the csi sidecar
		httpRes, err := http.Get(obj.Spec.CBTEndpoint)
		if err != nil {
			http.Error(resp, fmt.Sprintf("failed to fetch CBT entries: %s", err), http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(httpRes.Body)
		if err != nil {
			http.Error(resp, fmt.Sprintf("failed to read CBT entries: %s", err), http.StatusInternalServerError)
			return
		}

		resp.WriteHeader(http.StatusOK)
		if _, err := resp.Write(body); err != nil {
			klog.Error(err)
			return
		}
	}), nil
}

// NewConnectOptions returns an empty options object that will be used to pass
// options to the Connect method. If nil, then a nil options object is passed to
// Connect. It may return a bool and a string. If true, the value of the request
// path below the object will be included as the named string in the serialization
// of the runtime object.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Connecter
func (c *cbt) NewConnectOptions() (runtime.Object, bool, string) {
	return &v1alpha1.VolumeSnapshotDeltaOption{}, false, ""
}

// ConnectMethods returns the list of HTTP methods handled by Connect
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Connecter
func (c *cbt) ConnectMethods() []string {
	return []string{"GET"}
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (c *cbt) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	return &v1alpha1.VolumeSnapshotDeltaList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cbt.storage.k8s.io/v1alpha1",
			Kind:       "VolumeSnapshotDeltaList",
		},
		ListMeta: metav1.ListMeta{},
		Items: []v1alpha1.VolumeSnapshotDelta{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-delta",
					Namespace: "default",
				},
				Spec: v1alpha1.VolumeSnapshotDeltaSpec{
					BaseVolumeSnapshotName:   "base",
					TargetVolumeSnapshotName: "target",
					Mode:                     "block",
				},
				Status: v1alpha1.VolumeSnapshotDeltaStatus{
					Error: "",
				},
			},
		},
	}, nil
}

// 'label' selects on labels; 'field' selects on the object's fields. Not all fields
// are supported; an error should be returned if 'field' tries to select on a field that
// isn't supported. 'resourceVersion' allows for continuing/starting a watch at a
// particular version.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Watcher
func (c *cbt) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	if c.watch == nil {
		c.watch = make(chan watch.Event)
	}
	return watch.NewProxyWatcher(c.watch), nil
}

func writeResponse(resp http.ResponseWriter, data interface{}) {
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(body); err != nil {
		klog.Error(err)
		return
	}
}
