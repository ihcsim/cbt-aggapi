package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/ihcsim/cbt-aggapi/pkg/apis/cbt/v1alpha1"
	"github.com/ihcsim/cbt-aggapi/pkg/generated/cbt/clientset/versioned"
	cbtgrpc "github.com/ihcsim/cbt-aggapi/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var listenAddr = flag.String("listen", ":8080", "Address of the HTTP server")

func main() {
	if _, err := registerDriver(); err != nil {
		klog.Fatal(err)
	}

	grpcClient, err := grpcClient()
	if err != nil {
		klog.Fatal(err)
	}

	http.HandleFunc("/", func(res http.ResponseWriter, r *http.Request) {
		var (
			ctx, cancel = context.WithTimeout(context.Background(), time.Second*180)
			grpcReq     = &cbtgrpc.VolumeSnapshotDeltaRequest{}
		)
		defer cancel()

		grpcRes, err := grpcClient.ListVolumeSnapshotDeltas(ctx, grpcReq)
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

		writeResponse(res, blockDeltas)
	})

	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		klog.Fatal(err)
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
		if err == nil || errors.IsAlreadyExists(createErr) {
			klog.Infof("successfully registered CSI driver %s", endpoint.GetName())
			createErr = nil
			cancel()
			return
		}
		klog.Info("retry registering CSI driver %s: %s", endpoint.GetName(), err)
	}, retry)

	return created, createErr
}

func grpcClient() (cbtgrpc.VolumeSnapshotDeltaServiceClient, error) {
	grpcTarget, exists := os.LookupEnv("GRPC_TARGET")
	if !exists {
		grpcTarget = ":9779"
	}

	opts := grpc.WithTransportCredentials(insecure.NewCredentials())
	clientConn, err := grpc.Dial(grpcTarget, opts)
	if err != nil {
		return nil, err
	}

	return cbtgrpc.NewVolumeSnapshotDeltaServiceClient(clientConn), nil
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
