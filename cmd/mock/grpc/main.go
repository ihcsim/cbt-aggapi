package main

import (
	"flag"
	"log"
	"net"

	pb "github.com/ihcsim/cbt-aggapi/pkg/grpc"
	grpcserver "github.com/ihcsim/cbt-aggapi/pkg/grpc/server"

	"google.golang.org/grpc"
)

var csiAddress = flag.String("csi-address", "/run/csi/socket", "Address of the CSI driver socket.")

func main() {
	flag.Parse()

	log.Printf("listening at %s", *csiAddress)
	listener, err := net.Listen("unix", *csiAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterVolumeSnapshotDeltaServiceServer(grpcServer, grpcserver.New())
	if err := grpcServer.Serve(listener); err != nil {
		log.Println(err)
	}
}
