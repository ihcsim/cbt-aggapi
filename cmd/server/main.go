package main

import (
	"flag"
	"log"
	"net"

	pb "github.com/ihcsim/cbt-controller/pkg/grpc"
	grpcserver "github.com/ihcsim/cbt-controller/pkg/grpc/server"

	"google.golang.org/grpc"
)

var listenAddr = flag.String("target", ":9779", "Address of the GRPC server")

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
