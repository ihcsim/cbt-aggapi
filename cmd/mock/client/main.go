package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/storage"

	"github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	"github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

const defaultPort = 8080

var listenAddr = flag.String("listen-addr", fmt.Sprintf(":%d", defaultPort), "Address to listen at")

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err)
	}

	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	namespace := os.Getenv("NAMESPACE")
	if err := doCreate(clientset, namespace); err != nil {
		klog.Fatal(err)
	}

	server, err := newServer(namespace, *listenAddr, clientset)
	if err != nil {
		klog.Fatal(err)
	}

	klog.Infof("listening at %s", *listenAddr)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			klog.Error(err)
		}
	}
}

func doCreate(clientset versioned.Interface, namespace string) error {
	const namePrefix = "test-delta"

	var (
		err     error
		errChan = make(chan error)
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*180)
	defer cancel()

	go func() {
		err := <-errChan
		if err != nil {
			cancel()
		}
	}()

	wg := &sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()
			resource := &v1alpha1.VolumeSnapshotDelta{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-0%d", namePrefix, index),
					Namespace: namespace,
				},
				Spec: v1alpha1.VolumeSnapshotDeltaSpec{
					BaseVolumeSnapshotName:   "vol-snap-base",
					TargetVolumeSnapshotName: "vol-snap-target",
				},
			}

			klog.Infof("attempting to create %q", resource.GetName())
			created, err := clientset.CbtV1alpha1().VolumeSnapshotDeltas(namespace).Create(
				ctx, resource, metav1.CreateOptions{})
			if err != nil {
				if e, ok := err.(*storage.StorageError); ok {
					if e.Code == storage.ErrCodeKeyExists {
						klog.Infof("VolumeSnapshotDelta %q exists", resource.GetName())
						return
					}
				}
				errChan <- err
				return
			}
			klog.Infof("created VolumeSnapshotDelta %q", created.GetName())
		}(i)
	}

	wg.Wait()
	return err
}

func newServer(namespace, addr string, clientset versioned.Interface) (*http.Server, error) {
	serveMux, err := newServeMux(clientset, namespace)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:    addr,
		Handler: serveMux,
	}, nil
}

type serveMux struct {
	cbt       versioned.Interface
	namespace string
	*http.ServeMux
}

func newServeMux(clientset versioned.Interface, namespace string) (*serveMux, error) {
	s := &serveMux{
		cbt:       clientset,
		namespace: namespace,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/apis/", s.handle)
	s.ServeMux = mux

	return s, nil
}

func (s *serveMux) handle(res http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*180)
	defer cancel()

	// expect path to be apis/<resource_name>/[full]
	trimmed := strings.TrimFunc(r.URL.Path, func(r rune) bool {
		return r == os.PathSeparator
	})
	parts := strings.Split(trimmed, fmt.Sprintf("%c", os.PathSeparator))

	var (
		resourceName string
		fetchEntries = "normal"
	)

	if len(parts) >= 2 {
		resourceName = parts[1]
	}

	if len(parts) >= 3 && parts[2] == "extended" {
		fetchEntries = parts[2]
	}

	// send a LIST request if no resource name is provided
	if len(parts) == 1 {
		klog.Info("request sent: LIST")
		list, err := s.cbt.CbtV1alpha1().VolumeSnapshotDeltas(s.namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		writeResponse(res, list)
		return
	}

	var (
		obj *v1alpha1.VolumeSnapshotDelta
		err error
	)
	klog.Infof("request sent: GET '%s/%s' (%s)", s.namespace, resourceName, fetchEntries)
	if fetchEntries == "extended" {
		result := s.cbt.CbtV1alpha1().RESTClient().Get().
			Resource("volumesnapshotdeltas").
			Name(resourceName).
			Namespace(s.namespace).
			Param("fetchcbd", "true").
			Param("limit", "250").
			Param("offset", "0").
			Do(ctx)
		if err := result.Error(); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		obj = &v1alpha1.VolumeSnapshotDelta{}
		if err := result.Into(obj); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		obj, err = s.cbt.CbtV1alpha1().VolumeSnapshotDeltas(s.namespace).Get(ctx, resourceName, metav1.GetOptions{})
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	klog.Infof("received object: %q", obj.GetName())

	writeResponse(res, obj)
	return
}

func writeResponse(res http.ResponseWriter, obj interface{}) {
	data, err := json.Marshal(obj)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if _, err := res.Write(data); err != nil {
		klog.Error(err)
		return
	}
}
