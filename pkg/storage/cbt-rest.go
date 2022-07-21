package storage

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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	restregistry "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog"
	builderresource "sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

var _ rest.Connecter = &cbt{}
var _ rest.CreaterUpdater = &cbt{}
var _ rest.GracefulDeleter = &cbt{}
var _ rest.Watcher = &cbt{}
var _ rest.Lister = &cbt{}
var _ rest.Scoper = &cbt{}

// NewCustomStorage creates a new instance of a custom storage provider used
// to handle changed block entries.
func NewCustomStorage(
	obj builderresource.Object,
	clientset cbtclient.Interface,
	etcdStorage storage.Interface,
) builderrest.ResourceHandlerProvider {
	return func(s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (restregistry.Storage, error) {
		return &cbt{
			clientset:       clientset,
			namespaceScoped: obj.NamespaceScoped(),
			newFunc:         obj.New,
			newListFunc:     obj.NewList,
			etcd:            etcdStorage,
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
	etcd            storage.Interface
	restregistry.TableConvertor
}

func (c *cbt) New() runtime.Object {
	return c.newFunc()
}

// NewList returns an empty object that can be used with the List call.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
func (c *cbt) NewList() runtime.Object {
	return c.newListFunc()
}

// NamespaceScoped returns true if the storage is namespaced
func (c *cbt) NamespaceScoped() bool {
	return c.namespaceScoped
}

// Connect returns an http.Handler that will handle the request/response for a given API invocation.
// The provided responder may be used for common API responses. The responder will write both status
// code and body, so the ServeHTTP method should exit after invoking the responder. The Handler will
// be used for a single API request and then discarded. The Responder is guaranteed to write to the
// same http.ResponseWriter passed to ServeHTTP.
func (c *cbt) Connect(ctx context.Context, id string, options runtime.Object, r restregistry.Responder) (http.Handler, error) {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		var (
			result  = v1alpha1.VolumeSnapshotDelta{}
			getOpts = storage.GetOptions{}
		)

		// retrieve object from etcd
		if err := c.etcd.Get(ctx, keyPath(id), getOpts, &result); err != nil {
			status := http.StatusInternalServerError
			if se, ok := err.(*storage.StorageError); ok && se.Code == storage.ErrCodeKeyNotFound {
				status = http.StatusNotFound
				err = se
			}
			http.Error(resp, fmt.Sprintf("can't find VolumeSnapshotDelta: %s", err), status)
			return
		}
		klog.Infof("found VolumeSnapshotDelta: %s", id)

		// parse query parameter options
		opts, ok := options.(*v1alpha1.VolumeSnapshotDeltaOption)
		if !ok {
			http.Error(resp, "failed to parse VolumeSnapshotDeltaOptions", http.StatusInternalServerError)
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
		endpoint := fmt.Sprintf("http://%s.%s:%d", obj.Spec.Service.Name, obj.Spec.Service.Namespace, obj.Spec.Service.Port)
		httpRes, err := http.Get(endpoint)
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
func (c *cbt) NewConnectOptions() (runtime.Object, bool, string) {
	return &v1alpha1.VolumeSnapshotDeltaOption{}, false, ""
}

// ConnectMethods returns the list of HTTP methods handled by Connect
func (c *cbt) ConnectMethods() []string {
	return []string{"GET"}
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (c *cbt) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	var (
		list v1alpha1.VolumeSnapshotDeltaList
		key  = v1alpha1.SchemeGroupResource.Group
		opts = storage.ListOptions{
			ResourceVersion:      options.ResourceVersion,
			ResourceVersionMatch: options.ResourceVersionMatch,
			Predicate: storage.SelectionPredicate{
				Label:               options.LabelSelector,
				Field:               options.FieldSelector,
				AllowWatchBookmarks: options.AllowWatchBookmarks,
				Limit:               options.Limit,
				Continue:            options.Continue,
				IndexLabels:         []string{},
				IndexFields:         []string{},
				GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
					return storage.DefaultNamespaceScopedAttr(obj)
				},
			},
		}
	)

	if err := c.etcd.List(ctx, key, opts, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// Create creates a new version of a resource.
func (c *cbt) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {

	casted, ok := obj.(*v1alpha1.VolumeSnapshotDelta)
	if !ok {
		return nil, fmt.Errorf("")
	}
	casted.SetCreationTimestamp(metav1.Now())

	var out v1alpha1.VolumeSnapshotDelta
	if err := c.etcd.Create(ctx, keyPath(casted.GetName()), casted, &out, 0); err != nil {
		return nil, err
	}
	klog.Infof("created VolumeSnapshotDelta: %s", out.GetName())

	return &out, nil
}

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (c *cbt) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {

	var (
		updated       v1alpha1.VolumeSnapshotDelta
		preconditions *storage.Preconditions
	)

	if objInfo.Preconditions() != nil {
		preconditions = &storage.Preconditions{
			UID:             objInfo.Preconditions().UID,
			ResourceVersion: objInfo.Preconditions().ResourceVersion,
		}
	}

	klog.Infof("updating VolumeSnapshotDelta: %s", name)
	if err := c.etcd.GuaranteedUpdate(
		ctx,
		keyPath(name),
		&updated,
		true,
		preconditions,
		func(input runtime.Object, res storage.ResponseMeta) (runtime.Object, *uint64, error) {
			newObj, err := objInfo.UpdatedObject(ctx, input)
			return newObj, nil, err
		},
		nil); err != nil {
		return nil, false, err
	}

	return &updated, false, nil
}

// Delete finds a resource in the storage and deletes it.
// The delete attempt is validated by the deleteValidation first.
// If options are provided, the resource will attempt to honor them or return an invalid
// request error.
// Although it can return an arbitrary error value, IsNotFound(err) is true for the
// returned error value err when the specified resource is not found.
// Delete *may* return the object that was deleted, or a status object indicating additional
// information about deletion.
// It also returns a boolean which is set to true if the resource was instantly
// deleted or false if it will be deleted asynchronously.
func (c *cbt) Delete(
	ctx context.Context,
	name string,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {

	var (
		out           v1alpha1.VolumeSnapshotDelta
		preconditions *storage.Preconditions
	)

	if options.Preconditions != nil {
		preconditions = &storage.Preconditions{
			UID:             options.Preconditions.UID,
			ResourceVersion: options.Preconditions.ResourceVersion,
		}
	}

	klog.Infof("deleting VolumeSnapshotDelta: %s", name)
	if err := c.etcd.Delete(
		ctx,
		keyPath(name),
		&out,
		preconditions,
		func(ctx context.Context, obj runtime.Object) error {
			return deleteValidation(ctx, obj)
		},
		nil); err != nil {
		return nil, false, err
	}

	return &out, false, nil
}

// 'label' selects on labels; 'field' selects on the object's fields. Not all fields
// are supported; an error should be returned if 'field' tries to select on a field that
// isn't supported. 'resourceVersion' allows for continuing/starting a watch at a
// particular version.
func (c *cbt) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	opts := storage.ListOptions{
		ResourceVersion:      options.ResourceVersion,
		ResourceVersionMatch: options.ResourceVersionMatch,
		Predicate: storage.SelectionPredicate{
			Label:               options.LabelSelector,
			Field:               options.FieldSelector,
			Limit:               options.Limit,
			Continue:            options.Continue,
			AllowWatchBookmarks: options.AllowWatchBookmarks,
		},
	}
	return c.etcd.Watch(ctx, v1alpha1.SchemeGroupResource.Group, opts)
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

func keyPath(key string) string {
	return fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, key)
}
