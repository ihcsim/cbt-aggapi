package storage

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ihcsim/cbt-controller/pkg/apis/cbt/v1alpha1"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

func NewStorageProvider(obj resource.Object) builderrest.ResourceHandlerProvider {
	return func(s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (registryrest.Storage, error) {
		return &custom{
			namespaceScoped: obj.NamespaceScoped(),
			newFunc:         obj.New,
			newListFunc:     obj.NewList,
			TableConvertor: registryrest.NewDefaultTableConvertor(schema.GroupResource{
				Group:    obj.GetGroupVersionResource().Group,
				Resource: obj.GetGroupVersionResource().Resource,
			}),
		}, nil
	}
}

type custom struct {
	namespaceScoped bool
	newFunc         func() runtime.Object
	newListFunc     func() runtime.Object
	watch           chan watch.Event
	registryrest.TableConvertor
}

func (m *custom) New() runtime.Object {
	return m.newFunc()
}

// NewList returns an empty object that can be used with the List call.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (m *custom) NewList() runtime.Object {
	return m.newListFunc()
}

// NamespaceScoped returns true if the storage is namespaced
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Scoper
func (m *custom) NamespaceScoped() bool {
	return m.namespaceScoped
}

// Connect returns an http.Handler that will handle the request/response for a given API invocation.
// The provided responder may be used for common API responses. The responder will write both status
// code and body, so the ServeHTTP method should exit after invoking the responder. The Handler will
// be used for a single API request and then discarded. The Responder is guaranteed to write to the
// same http.ResponseWriter passed to ServeHTTP.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Connecter
func (m *custom) Connect(ctx context.Context, id string, options runtime.Object, r registryrest.Responder) (http.Handler, error) {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		result := &v1alpha1.VolumeSnapshotDelta{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-delta",
				Namespace: "default",
			},
			Spec: v1alpha1.VolumeSnapshotDeltaSpec{
				BaseVolumeSnapshotName:   "base",
				TargetVolumeSnapshotName: "target",
			},
			Status: v1alpha1.VolumeSnapshotDeltaStatus{
				Error:       "",
				CallbackURL: "example.com",
			},
		}

		cast, ok := options.(*v1alpha1.VolumeSnapshotDeltaOption)
		if !ok {
			http.Error(res, "failed to read VolumeSnapshotDelta options", http.StatusInternalServerError)
			return
		}

		if cast.FetchCBD {
			result.Status.ChangedBlockDeltas = []*v1alpha1.ChangedBlockDelta{
				{
					Offset:         1,
					BlockSizeBytes: 1024000,
				},
				{
					Offset:         2,
					BlockSizeBytes: 1024000,
				},
				{
					Offset:         3,
					BlockSizeBytes: 1024000,
				},
			}
		}

		body, err := json.Marshal(result)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
		if _, err := res.Write(body); err != nil {
			log.Println(err)
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
func (m *custom) NewConnectOptions() (runtime.Object, bool, string) {
	return &v1alpha1.VolumeSnapshotDeltaOption{}, false, ""
}

// ConnectMethods returns the list of HTTP methods handled by Connect
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Connecter
func (m *custom) ConnectMethods() []string {
	return []string{"GET"}
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (m *custom) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	return &v1alpha1.VolumeSnapshotDeltaList{}, nil
}

// Create creates a new version of a resource.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Create
func (m *custom) Create(ctx context.Context, obj runtime.Object, createValidation registryrest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return &v1alpha1.VolumeSnapshotDelta{}, nil
}

// 'label' selects on labels; 'field' selects on the object's fields. Not all fields
// are supported; an error should be returned if 'field' tries to select on a field that
// isn't supported. 'resourceVersion' allows for continuing/starting a watch at a
// particular version.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Watcher
func (m *custom) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	if m.watch == nil {
		m.watch = make(chan watch.Event)
	}
	return watch.NewProxyWatcher(m.watch), nil
}
