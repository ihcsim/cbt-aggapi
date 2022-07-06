package v1alpha1

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Connect returns an http.Handler that will handle the request/response for a given API invocation.
// The provided responder may be used for common API responses. The responder will write both status
// code and body, so the ServeHTTP method should exit after invoking the responder. The Handler will
// be used for a single API request and then discarded. The Responder is guaranteed to write to the
// same http.ResponseWriter passed to ServeHTTP.
func (v *VolumeSnapshotDelta) Connect(
	ctx context.Context,
	id string,
	options runtime.Object,
	r rest.Responder) (http.Handler, error) {
	opts, ok := options.(*VolumeSnapshotDelta)
	if !ok {
		return nil, fmt.Errorf("invalid resource object: %+v", options)
	}
	fmt.Printf(">>> %+v\n", opts)

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

	}), nil
}

// NewConnectOptions returns an empty options object that will be used to pass
// options to the Connect method. If nil, then a nil options object is passed to
// Connect. It may return a bool and a string. If true, the value of the request
// path below the object will be included as the named string in the serialization
// of the runtime object.
func (v *VolumeSnapshotDelta) NewConnectOptions() (runtime.Object, bool, string) {
	return &VolumeSnapshotDelta{}, false, ""
}

// ConnectMethods returns the list of HTTP methods handled by Connect
func ConnectMethods() []string {
	return []string{"POST"}
}
