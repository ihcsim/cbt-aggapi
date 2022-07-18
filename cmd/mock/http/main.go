package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	"github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	cbtgrpc "github.com/ihcsim/cbt-aggapi/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var (
	listenAddr = flag.String("listen-addr", ":8080", "Address to listen at")
	grpcTarget = flag.String("grpc-target", ":9779", "Address of the GRPC server")
)

func main() {
	flag.Parse()

	if _, err := registerDriver(); err != nil {
		klog.Fatal(err)
	}

	server, err := newServer()
	if err != nil {
		klog.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		klog.Info("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			klog.Error(err)
		}
	}()

	klog.Info("listening at: ", *listenAddr)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			klog.Error(err)
		}
	}
}

func registerDriver() (runtime.Object, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	var (
		endpoint = &v1alpha1.DriverDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name: os.Getenv("CSI_DRIVER_NAME"),
			},
			Spec: v1alpha1.DriverDiscoverySpec{
				Driver:      os.Getenv("CSI_DRIVER_NAME"),
				CBTEndpoint: os.Getenv("CSI_CBT_ENDPOINT"),
			},
		}
		retry       = time.Second * 5
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute*5)

		created   runtime.Object
		createErr error
	)

	klog.Infof("registering CSI driver %s", endpoint.GetName())
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		created, createErr = clientset.CbtV1alpha1().DriverDiscoveries().Create(ctx,
			endpoint,
			metav1.CreateOptions{})
		if err == nil || apierrors.IsAlreadyExists(createErr) {
			klog.Infof("successfully registered CSI driver %s", endpoint.GetName())
			createErr = nil
			cancel()
			return
		}
		klog.Info("retry registering CSI driver %s: %s", endpoint.GetName(), err)
	}, retry)

	return created, createErr
}

func newServer() (*http.Server, error) {
	serveMux, err := newServeMux(*grpcTarget)
	if err != nil {
		klog.Fatal(err)
	}

	return &http.Server{
		Addr:    *listenAddr,
		Handler: serveMux,
	}, nil
}

type serveMux struct {
	grpc *grpc.ClientConn
	*http.ServeMux
}

func newServeMux(grpcTarget string) (*serveMux, error) {
	klog.Infof("connecting to GRPC target at %s", grpcTarget)
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())
	clientConn, err := grpc.Dial(grpcTarget, opts)
	if err != nil {
		return nil, err
	}

	s := &serveMux{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)

	s.grpc = clientConn
	s.ServeMux = mux
	return s, nil
}

func (s *serveMux) handle(res http.ResponseWriter, r *http.Request) {
	var (
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*180)
		grpcReq     = &cbtgrpc.VolumeSnapshotDeltaRequest{}
	)
	defer cancel()

	client := cbtgrpc.NewVolumeSnapshotDeltaServiceClient(s.grpc)
	grpcRes, err := client.ListVolumeSnapshotDeltas(ctx, grpcReq)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	blockDeltas := []*v1alpha1.ChangedBlockDelta{}
	for _, cbd := range grpcRes.GetBlockDelta().GetChangedBlockDeltas() {
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

	body, err := json.Marshal(blockDeltas)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if _, err := res.Write(body); err != nil {
		klog.Error(err)
		return
	}
}
