package storage

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ihcsim/cbt-controller/pkg/apis/cbt/v1alpha1"
	"github.com/ihcsim/cbt-controller/pkg/grpc"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorageProvider(
	obj resource.Object,
	grpcClient grpc.VolumeSnapshotDeltaServiceClient,
) builderrest.ResourceHandlerProvider {
	return func(s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (registryrest.Storage, error) {
		return &custom{
			grpc:            grpcClient,
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
	grpc            grpc.VolumeSnapshotDeltaServiceClient
	k8sClient       client.Client
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

		var (
			ctx, cancel = context.WithTimeout(context.Background(), time.Second*180)
			grpcReq     = &grpc.VolumeSnapshotDeltaRequest{}
		)
		defer cancel()

		grpcResp, err := m.grpc.ListVolumeSnapshotDeltas(ctx, grpcReq)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		blockDeltas := []*v1alpha1.ChangedBlockDelta{}
		for _, cbd := range grpcResp.GetBlockDelta().GetChangedBlockDeltas() {
			blockDeltas = append(blockDeltas, &v1alpha1.ChangedBlockDelta{
				Offset:         cbd.GetOffset(),
				BlockSizeBytes: cbd.GetBlockSizeBytes(),
				DataToken: v1alpha1.DataToken{
					Token:        cbd.GetDataToken().GetToken(),
					IssuanceTime: metav1.NewTime(cbd.GetDataToken().GetIssuanceTime().AsTime()),
					TTL: metav1.Duration{
						Duration: cbd.GetDataToken().GetTtlSeconds().AsDuration(),
					},
				},
			})
			klog.Infof("found changed block at offset %d (%d)\n", cbd.GetOffset(), cbd.GetBlockSizeBytes())
		}

		result.Status.ChangedBlockDeltas = blockDeltas
		writeResponse(resp, result)
	}), nil
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

// Create creates a new version of a resource.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Create
func (m *custom) Create(ctx context.Context, obj runtime.Object, createValidation registryrest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	o, ok := obj.(*v1alpha1.VolumeSnapshotDelta)
	if !ok {
		return nil, fmt.Errorf("failed to create resource")
	}

	opts := &client.CreateOptions{
		Raw: options,
	}

	if err := m.k8sClient.Create(ctx, o, opts); err != nil {
		return nil, err
	}

	return o, nil
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
