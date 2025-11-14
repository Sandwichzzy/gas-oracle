package main

import (
	"context"
	"fmt"

	gas_oracle "github.com/Sandwichzzy/gas-oracle"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/Sandwichzzy/gas-oracle/common/cliapp"
	"github.com/Sandwichzzy/gas-oracle/common/opio"
	"github.com/Sandwichzzy/gas-oracle/config"
	"github.com/Sandwichzzy/gas-oracle/database"
	grpc2 "github.com/Sandwichzzy/gas-oracle/services/grpc"
)

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Value:   "./gas-oracle.yaml",
		Aliases: []string{"c"},
		Usage:   "path to config file",
		EnvVars: []string{"GAS_ORACLE_CONFIG"},
	}
	MigrationsFlag = &cli.StringFlag{
		Name:    "migrations-dir",
		Value:   "./migrations",
		Usage:   "path to migrations folder",
		EnvVars: []string{"GAS_ORACLE_MIGRATIONS_DIR"},
	}
)

func runOracle(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log.Info("running gas oracle...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return nil, err
	}
	return gas_oracle.NewGasOracle(ctx.Context, cfg, shutdown)
}

func runGRPCSever(ctx *cli.Context, _ context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	fmt.Println("running grpc services...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("config error", "err", err)
		return nil, err
	}
	grpcServerCfg := &grpc2.TokenPriceRpcConfig{
		Host: cfg.Server.Host,
		Port: cfg.Server.Port,
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDb)
	if err != nil {
		log.Error("new database fail", "err", err)
	}
	return grpc2.NewTokenPriceRpcService(grpcServerCfg, db)
}

func runMigrations(ctx *cli.Context) error {
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("running migrations...")
	cfg, err := config.New(ctx.String(ConfigFlag.Name))
	if err != nil {
		log.Error("failed to load config", "err", err)
		return err
	}
	db, err := database.NewDB(ctx.Context, cfg.MasterDb)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			return
		}
	}(db)
	err = db.ExecuteSQLMigration(ctx.String(MigrationsFlag.Name))
	if err != nil {
		return err
	}
	log.Info("running migrations and create table from template success")
	return nil
}

func NewCli() *cli.App {
	flags := []cli.Flag{ConfigFlag}
	migrationFlags := []cli.Flag{MigrationsFlag, ConfigFlag}
	return &cli.App{
		Version:              "v0.0.1",
		Description:          "an indexer bridge gas oracle with a grpc server",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "grpc",
				Flags:       flags,
				Description: "Runs the gprc service",
				Action:      cliapp.LifecycleCmd(runGRPCSever),
			},
			{
				Name:        "index",
				Flags:       flags,
				Description: "Runs the indexing service",
				Action:      cliapp.LifecycleCmd(runOracle),
			},
			{
				Name:        "migrate",
				Flags:       migrationFlags,
				Description: "Runs the database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "version",
				Description: "print version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
