package main

import (
	"flag"
	"log"
	"net"

	pb "github.com/ihcsim/cbt-aggapi/pkg/grpc"
	grpcserver "github.com/ihcsim/cbt-aggapi/pkg/grpc/server"

	"google.golang.org/grpc"
)

var listenAddr = flag.String("listen", ":9779", "Address of the GRPC server")

func main() {
	flag.Parse()

	log.Printf("listening at %s", *listenAddr)
	listener, err := net.Listen("tcp", *listenAddr)
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
