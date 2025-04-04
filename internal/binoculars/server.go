package binoculars

import (
	"os"
	"sync"

	"github.com/armadaproject/armada/internal/binoculars/configuration"
	"github.com/armadaproject/armada/internal/binoculars/server"
	"github.com/armadaproject/armada/internal/binoculars/service"
	"github.com/armadaproject/armada/internal/common/auth"
	"github.com/armadaproject/armada/internal/common/cluster"
	grpcCommon "github.com/armadaproject/armada/internal/common/grpc"
	log "github.com/armadaproject/armada/internal/common/logging"
	"github.com/armadaproject/armada/pkg/api/binoculars"
)

func StartUp(config *configuration.BinocularsConfig) (func(), *sync.WaitGroup) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	kubernetesClientProvider, err := cluster.NewKubernetesClientProvider(
		config.ImpersonateUsers,
		config.Kubernetes.QPS,
		config.Kubernetes.Burst,
	)
	if err != nil {
		log.Errorf("Failed to connect to kubernetes because %s", err)
		os.Exit(-1)
	}

	authServices, err := auth.ConfigureAuth(config.Auth)
	if err != nil {
		log.Errorf("Failed to create auth services %s", err)
		os.Exit(-1)
	}

	grpcServer := grpcCommon.CreateGrpcServer(config.Grpc.KeepaliveParams, config.Grpc.KeepaliveEnforcementPolicy, authServices, config.Grpc.Tls)

	permissionsChecker := auth.NewPrincipalPermissionChecker(
		config.Auth.PermissionGroupMapping,
		config.Auth.PermissionScopeMapping,
		config.Auth.PermissionClaimMapping,
	)

	logService := service.NewKubernetesLogService(kubernetesClientProvider)
	cordonService := service.NewKubernetesCordonService(config.Cordon, permissionsChecker, kubernetesClientProvider)
	binocularsServer := server.NewBinocularsServer(logService, cordonService)
	binoculars.RegisterBinocularsServer(grpcServer, binocularsServer)

	grpcCommon.Listen(config.GrpcPort, grpcServer, wg)

	return grpcServer.GracefulStop, wg
}
