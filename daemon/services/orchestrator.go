package services

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/jbrodriguez/controlrd/daemon/domain"
	"github.com/jbrodriguez/controlrd/daemon/logger"
	"github.com/jbrodriguez/controlrd/daemon/services/api"
)

type Orchestrator struct {
	ctx *domain.Context
}

func CreateOrchestrator(ctx *domain.Context) *Orchestrator {
	return &Orchestrator{
		ctx: ctx,
	}
}

func (o *Orchestrator) Run() error {
	logger.Blue("starting controlr %s ...", o.ctx.Version)

	api := api.Create(o.ctx)

	err := api.Run()
	if err != nil {
		return err
	}

	w := make(chan os.Signal, 1)
	signal.Notify(w, syscall.SIGTERM, syscall.SIGINT)
	logger.Blue("received %s signal. shutting down the app ...", <-w)

	return nil
}
