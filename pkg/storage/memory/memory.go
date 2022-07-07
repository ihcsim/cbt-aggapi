package memory

import (
	"context"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

func NewStorageProvider(obj resource.Object) builderrest.ResourceHandlerProvider {
	return func(s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (registryrest.Storage, error) {
		return &memory{
			namespaceScoped: obj.NamespaceScoped(),
			newFunc:         obj.New,
			newListFunc:     obj.NewList,
		}, nil
	}
}

type memory struct {
	namespaceScoped bool
	newFunc         func() runtime.Object
	newListFunc     func() runtime.Object
}

func (m *memory) New() runtime.Object {
	return m.newFunc()
}

// NewList returns an empty object that can be used with the List call.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (m *memory) NewList() runtime.Object {
	return m.newListFunc()
}

// NamespaceScoped returns true if the storage is namespaced
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Scoper
func (m *memory) NamespaceScoped() bool {
	return m.namespaceScoped
}

// Get finds a resource in the storage by name and returns it.
// Although it can return an arbitrary error value, IsNotFound(err) is true for the
// returned error value err when the specified resource is not found.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Getter
func (m *memory) Get(
	ctx context.Context,
	name string,
	options *metav1.GetOptions,
) (runtime.Object, error) {
	return nil, nil
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (m *memory) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	return nil, nil
}

// TableConvertor ensures all list implementers also implement table conversion
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Lister
func (m *memory) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return nil, nil
}

// Create creates a new version of a resource.
// See https://pkg.go.dev/k8s.io/apiserver/pkg/registry/rest#Creater
func (m *memory) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation registryrest.ValidateObjectFunc,
	options *metav1.CreateOptions,
) (runtime.Object, error) {
	return nil, nil
}
