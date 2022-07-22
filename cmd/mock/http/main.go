package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	"github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	cbtgrpc "github.com/ihcsim/cbt-aggapi/pkg/grpc"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/metrics"
	"google.golang.org/grpc"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

const defaultPort = 8080

var (
	listenAddr = flag.String("listen-addr", fmt.Sprintf(":%d", defaultPort), "Address to listen at")
	csiAddress = flag.String("csi-address", "/run/csi/socket", "Address of the CSI driver socket.")
)

func main() {
	flag.Parse()

	var (
		driver    = os.Getenv("CSI_DRIVER_NAME")
		svc       = os.Getenv("SVC_NAME")
		namespace = os.Getenv("SVC_NAMESPACE")
	)
	svcPort, err := strconv.ParseInt(os.Getenv("SVC_PORT"), 10, 32)
	if err != nil {
		klog.Error(err)
		svcPort = defaultPort
	}

	if _, err := registerDriver(driver, svc, namespace, svcPort); err != nil {
		klog.Fatal(err)
	}

	server, err := newServer(driver, *csiAddress)
	if err != nil {
		klog.Fatal(err)
	}
	notifyShutdown(server)

	klog.Infof("listening at: %s", *listenAddr)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			klog.Error(err)
		}
	}
}

func registerDriver(driver, svc, namespace string, svcPort int64) (runtime.Object, error) {
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
				Name: driver,
			},
			Spec: v1alpha1.DriverDiscoverySpec{
				Driver: driver,
				Service: v1alpha1.Service{
					Name:      svc,
					Namespace: namespace,
					Path:      "/",
					Port:      svcPort,
				},
			},
		}
		retry       = time.Second * 5
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute*5)
		created     runtime.Object
	)

	klog.Infof("registering CSI driver %s", endpoint.GetName())
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		created, err = clientset.CbtV1alpha1().DriverDiscoveries().Create(ctx,
			endpoint,
			metav1.CreateOptions{})
		if err == nil || apierrors.IsAlreadyExists(err) {
			klog.Infof("successfully registered CSI driver %s", endpoint.GetName())
			err = nil
			cancel()
			return
		}
		klog.Infof("retry registering CSI driver %s: %s", endpoint.GetName(), err)
	}, retry)

	return created, err
}

func newServer(driver, grpcTarget string) (*http.Server, error) {
	serveMux, err := newServeMux(driver, grpcTarget)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:    *listenAddr,
		Handler: serveMux,
	}, nil
}

func notifyShutdown(server *http.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		klog.Info("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			klog.Error(err)
		}
	}()
}

type serveMux struct {
	grpc *grpc.ClientConn
	*http.ServeMux
}

func newServeMux(driverName, grpcTarget string) (*serveMux, error) {
	klog.Infof("connecting to CSI driver at %s", grpcTarget)
	metricsManager := metrics.NewCSIMetricsManager(driverName)
	csiConn, err := connection.Connect(
		grpcTarget,
		metricsManager,
		connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
	if err != nil {
		return nil, err
	}

	s := &serveMux{
		grpc: csiConn,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
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
