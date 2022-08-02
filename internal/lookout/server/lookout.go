package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/G-Research/armada/internal/lookout/repository"
	"github.com/G-Research/armada/pkg/api"
	"github.com/G-Research/armada/pkg/api/lookout"
	"github.com/G-Research/armada/pkg/client"
)

type LookoutServer struct {
	armadaUrl     string
	jobRepository repository.JobRepository
}

func NewLookoutServer(armadaUrl string, jobRepository repository.JobRepository) *LookoutServer {
	return &LookoutServer{
		armadaUrl:     armadaUrl,
		jobRepository: jobRepository,
	}
}

func (s *LookoutServer) Overview(ctx context.Context, _ *types.Empty) (*lookout.SystemOverview, error) {
	queues, err := s.jobRepository.GetQueueInfos(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query queue stats: %s", err)
	}
	return &lookout.SystemOverview{Queues: queues}, nil
}

func (s *LookoutServer) GetJobSets(ctx context.Context, opts *lookout.GetJobSetsRequest) (*lookout.GetJobSetsResponse, error) {
	jobSets, err := s.jobRepository.GetJobSetInfos(ctx, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query queue stats: %s", err)
	}
	return &lookout.GetJobSetsResponse{JobSetInfos: jobSets}, nil
}

func (s *LookoutServer) GetJobs(ctx context.Context, opts *lookout.GetJobsRequest) (*lookout.GetJobsResponse, error) {
	jobInfos, err := s.jobRepository.GetJobs(ctx, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query jobs in queue: %s", err)
	}
	return &lookout.GetJobsResponse{JobInfos: jobInfos}, nil
}

func (s *LookoutServer) CancelJobSet(ctx context.Context, req *lookout.CancelJobSetRequest) (*lookout.CancelJobResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	fmt.Println(ok)
	result := md.Get("Authorization")
	log.Infof("Length of Authorization %d", len(result))
	result = md.Get("authorization")
	log.Infof("Length of Authorization %d", len(result))

	//outgoingContext := metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"authorization": "test"}))

	apiConnectionDetails := client.ApiConnectionDetails{
		ArmadaUrl: s.armadaUrl,
	}

	err := client.WithSubmitClient(&apiConnectionDetails, func(submitClient api.SubmitClient) error {
		result, err := submitClient.CancelJobs(ctx, &api.JobCancelRequest{
			JobId:    "",
			JobSetId: req.JobSet,
			Queue:    req.Queue,
		})
		if err != nil {
			log.Errorf("Failed to cancel jobset %s becaues %s", req.JobSet, err)
			return err
		}

		log.Infof("Cancelled %s", strings.Join(result.CancelledIds, ","))
		return err
	})
	return new(lookout.CancelJobResponse), err
}
