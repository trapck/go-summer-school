package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"wasfaty.api/pkg/cerror"
	"wasfaty.api/pkg/env"
	"wasfaty.api/pkg/log"
	"wasfaty.api/pkg/log/logger"
	"wasfaty.api/services/mpi/adapter/api/extdocregistry"
	"wasfaty.api/services/mpi/adapter/api/fhir"
	"wasfaty.api/services/mpi/adapter/api/otp"
	"wasfaty.api/services/mpi/controller/http"
	"wasfaty.api/services/mpi/usecase"
)

func runService(ctx context.Context) error {
	cfg := new(config)
	if err := env.ParseCfg(cfg); err != nil {
		return err
	}

	log.SetGlobalLogLevel(cfg.LogLevel)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fc := fhir.NewClient(&cfg.FHIR)
	oc := otp.NewClient(&cfg.OTPClient)
	edrc := extdocregistry.NewClient()

	uc := usecase.New(fc, oc, edrc)
	s := http.NewServer(cfg.HTTPServer, cfg.Service, &cfg.Trace, uc)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT)

	select {
	case err := <-s.ListenAndServe(ctx):
		return cerror.New(ctx, cerror.KindInternal, err).LogError()
	case sig := <-sigCh:
		err := s.Shutdown()
		if err != nil {
			return cerror.New(ctx, cerror.KindInternal, err).LogError()
		}

		log.Log(logger.NewEventF(ctx, logger.LevelInfo, "terminating got [%v] signal", sig))

		return nil
	}
}
