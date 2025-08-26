package rewardpool_grpc_service

import (
	"context"
	"io"
	"net"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ActorSystem is an interface that actor.System implements.
type ActorSystem interface {
	State() []types.PoolReward
	Draw() <-chan actor.DrawResponse
	Stop()
	UpdateItem(id string, quantity int, weight int64) error
	GetRequestID() uint64
	SetRequestID(id uint64)
}

// RewardPoolService is a gRPC service that exposes the reward pool functionality.
type RewardPoolService struct {
	UnimplementedRewardPoolServiceServer
	system ActorSystem
}

// NewRewardPoolService creates a new RewardPoolService.
func NewRewardPoolService(system ActorSystem) *RewardPoolService {
	return &RewardPoolService{
		system: system,
	}
}

// ListenAndServe starts the gRPC server.
func ListenAndServe(ctx context.Context, system ActorSystem, listenAddress string) error {
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	grpcService := NewRewardPoolService(system)
	RegisterRewardPoolServiceServer(s, grpcService)

	// Addon: support grpc-cli or grpccurl list
	// Register reflection service on gRPC server.
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	return s.Serve(lis)
}

// GetState returns the current state of the reward pool.
func (s *RewardPoolService) GetState(ctx context.Context, req *GetStateRequest) (*GetStateResponse, error) {
	state := s.system.State()
	items := make([]*RewardItem, 0, len(state))
	for _, item := range state {
		items = append(items, &RewardItem{
			ItemId:      item.ItemID,
			Quantity:    int32(item.Quantity),
			Probability: item.Probability,
		})
	}
	return &GetStateResponse{
		Items: items,
	}, nil
}

// Draw draws items from the reward pool.
func (s *RewardPoolService) Draw(stream RewardPoolService_DrawServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		count := req.GetCount()
		if count <= 0 {
			count = 1
		}

		for i := 0; i < int(count); i++ {
			resp := <-s.system.Draw()
			var errMsg string
			if resp.Err != nil {
				errMsg = resp.Err.Error()
			}
			if err := stream.Send(&DrawResponse{
				RequestId: resp.RequestID,
				ItemId:    resp.Item,
				Error:     errMsg,
			}); err != nil {
				return err
			}
		}
	}
}
