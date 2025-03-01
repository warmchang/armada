package leader

import (
	"fmt"
	"strings"
	"sync"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/armadaproject/armada/internal/scheduler/configuration"
	"github.com/armadaproject/armada/pkg/client"
)

const leaseHolderNameToken = "<name>"

type LeaderClientConnectionProvider interface {
	GetCurrentLeaderClientConnection() (bool, *grpc.ClientConn, error)
}

type LeaderConnectionProvider struct {
	leaderController LeaderController
	leaderConfig     configuration.LeaderConfig
	connectionLock   sync.Mutex
	connectionByName map[string]*grpc.ClientConn
	metrics          *grpc_prometheus.ClientMetrics
}

func NewLeaderConnectionProvider(leaderController LeaderController, leaderConfig configuration.LeaderConfig, metrics *grpc_prometheus.ClientMetrics) *LeaderConnectionProvider {
	return &LeaderConnectionProvider{
		leaderController: leaderController,
		leaderConfig:     leaderConfig,
		connectionLock:   sync.Mutex{},
		connectionByName: map[string]*grpc.ClientConn{},
		metrics:          metrics,
	}
}

func (l *LeaderConnectionProvider) GetCurrentLeaderClientConnection() (bool, *grpc.ClientConn, error) {
	currentLeader := l.leaderController.GetLeaderReport()

	if currentLeader.IsCurrentProcessLeader {
		return true, nil, nil
	}
	if currentLeader.LeaderName == "" {
		return false, nil, fmt.Errorf("no leader found to retrieve scheduling reports from")
	}

	leaderClient, err := l.getClientByName(currentLeader.LeaderName)
	return false, leaderClient, err
}

func (l *LeaderConnectionProvider) getClientByName(currentLeaderName string) (*grpc.ClientConn, error) {
	l.connectionLock.Lock()
	defer l.connectionLock.Unlock()

	if leaderClient, present := l.connectionByName[currentLeaderName]; present {
		return leaderClient, nil
	}

	leaderConnectionDetails := l.leaderConfig.LeaderConnection
	leaderConnectionDetails.ArmadaUrl = strings.ReplaceAll(leaderConnectionDetails.ArmadaUrl, leaseHolderNameToken, currentLeaderName)

	apiConnection, err := createApiConnection(leaderConnectionDetails, l.metrics)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating connection to leader")
	}

	l.connectionByName[currentLeaderName] = apiConnection
	return apiConnection, nil
}

func createApiConnection(connectionDetails client.ApiConnectionDetails, clientMetrics *grpc_prometheus.ClientMetrics) (*grpc.ClientConn, error) {
	return client.CreateApiConnectionWithCallOptions(
		&connectionDetails,
		[]grpc.CallOption{},
		grpc.WithChainUnaryInterceptor(clientMetrics.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(clientMetrics.StreamClientInterceptor()),
	)
}
