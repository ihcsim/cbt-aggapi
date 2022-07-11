package server

import (
	"context"
	"time"

	pb "github.com/ihcsim/cbt-controller/pkg/grpc"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedVolumeSnapshotDeltaServiceServer
}

func New() *Server {
	return &Server{}
}

func (s *Server) ListVolumeSnapshotDeltas(
	ctx context.Context,
	req *pb.VolumeSnapshotDeltaRequest,
) (*pb.VolumeSnapshotDeltaResponse, error) {
	var (
		blockDeltas = []*pb.ChangedBlockDelta{
			{
				Offset:         0,
				BlockSizeBytes: 524288,
				DataToken: &pb.DataToken{
					Token:        "ieEEQ9Bj7E6XR",
					IssuanceTime: timestamppb.Now(),
					TtlSeconds:   durationpb.New(time.Minute * 180),
				},
			},
			{
				Offset:         1,
				BlockSizeBytes: 524288,
				DataToken: &pb.DataToken{
					Token:        "widvSdPYZCyLB",
					IssuanceTime: timestamppb.Now(),
					TtlSeconds:   durationpb.New(time.Minute * 180),
				},
			},
			{
				Offset:         2,
				BlockSizeBytes: 524288,
				DataToken: &pb.DataToken{
					Token:        "VtSebH83xYzvB",
					IssuanceTime: timestamppb.Now(),
					TtlSeconds:   durationpb.New(time.Minute * 180),
				},
			},
		}
		nextToken       = "uXonK48vfznJS"
		volumeSizeBytes = uint64(1073741824)
	)

	return &pb.VolumeSnapshotDeltaResponse{
		BlockDelta: &pb.BlockVolumeSnapshotDelta{
			ChangedBlockDeltas: blockDeltas,
		},
		VolumeSizeBytes: &volumeSizeBytes,
		NextToken:       &nextToken,
	}, nil
}
