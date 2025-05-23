package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/utils/clock"

	"github.com/armadaproject/armada/internal/common"
	"github.com/armadaproject/armada/internal/common/armadacontext"
	"github.com/armadaproject/armada/internal/common/database"
	log "github.com/armadaproject/armada/internal/common/logging"
	"github.com/armadaproject/armada/internal/common/profiling"
	"github.com/armadaproject/armada/internal/lookout"
	"github.com/armadaproject/armada/internal/lookout/configuration"
	"github.com/armadaproject/armada/internal/lookout/gen/restapi"
	"github.com/armadaproject/armada/internal/lookout/pruner"
	"github.com/armadaproject/armada/internal/lookout/schema"
	armada_config "github.com/armadaproject/armada/internal/server/configuration"
)

const (
	CustomConfigLocation string = "config"
	MigrateDatabase             = "migrateDatabase"
	PruneDatabase               = "pruneDatabase"
)

func init() {
	pflag.StringSlice(
		CustomConfigLocation,
		[]string{},
		"Fully qualified path to application configuration file (for multiple config files repeat this arg or separate paths with commas)",
	)
	pflag.Bool(MigrateDatabase, false, "Migrate database instead of running server")
	pflag.Bool(PruneDatabase, false, "Prune database of old jobs instead of running server")
	pflag.Parse()
}

func makeContext() (*armadacontext.Context, func()) {
	ctx := armadacontext.Background()
	ctx, cancel := armadacontext.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, func() {
		signal.Stop(c)
		cancel()
	}
}

func migrate(ctx *armadacontext.Context, config configuration.LookoutConfig) {
	db, err := database.OpenPgxPool(config.Postgres)
	if err != nil {
		panic(err)
	}

	migrations, err := schema.LookoutMigrations()
	if err != nil {
		panic(err)
	}

	err = database.UpdateDatabase(ctx, db, migrations)
	if err != nil {
		panic(err)
	}
}

func prune(ctx *armadacontext.Context, config configuration.LookoutConfig) {
	var dbConfig armada_config.PostgresConfig
	if config.PrunerConfig.Postgres.Connection != nil {
		dbConfig = config.PrunerConfig.Postgres
	} else {
		dbConfig = config.Postgres
	}

	db, err := database.OpenPgxConn(dbConfig)
	if err != nil {
		panic(err)
	}

	if config.PrunerConfig.Timeout <= 0 {
		panic("timeout must be greater than 0")
	}
	if config.PrunerConfig.ExpireAfter <= 0 {
		panic("expireAfter must be greater than 0")
	}
	if config.PrunerConfig.BatchSize <= 0 {
		panic("batchSize must be greater than 0")
	}
	log.Infof("expireAfter: %v, batchSize: %v, timeout: %v",
		config.PrunerConfig.ExpireAfter, config.PrunerConfig.BatchSize, config.PrunerConfig.Timeout)

	ctxTimeout, cancel := armadacontext.WithTimeout(ctx, config.PrunerConfig.Timeout)
	defer cancel()
	err = pruner.PruneDb(
		ctxTimeout,
		db,
		config.PrunerConfig.ExpireAfter,
		config.PrunerConfig.DeduplicationExpireAfter,
		config.PrunerConfig.BatchSize,
		clock.RealClock{})
	if err != nil {
		panic(err)
	}
}

func main() {
	log.MustConfigureApplicationLogging()
	common.BindCommandlineArguments()

	var config configuration.LookoutConfig
	userSpecifiedConfigs := viper.GetStringSlice(CustomConfigLocation)
	common.LoadConfig(&config, "./config/lookout", userSpecifiedConfigs)

	// Expose profiling endpoints if enabled.
	err := profiling.SetupPprof(config.Profiling, armadacontext.Background(), nil)
	if err != nil {
		log.Fatalf("Pprof setup failed, exiting, %v", err)
	}

	ctx, cleanup := makeContext()
	defer cleanup()

	if viper.GetBool(MigrateDatabase) {
		log.Info("Migrating database")
		migrate(ctx, config)
		return
	}

	if viper.GetBool(PruneDatabase) {
		log.Info("Pruning database")
		prune(ctx, config)
		return
	}

	restapi.UIConfig = config.UIConfig

	if err := lookout.Serve(config); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
