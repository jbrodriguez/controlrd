package cmd

import (
	"github.com/jbrodriguez/controlrd/daemon/domain"
	"github.com/jbrodriguez/controlrd/daemon/services"
)

type Boot struct{}

func (b *Boot) Run(ctx *domain.Context) error {
	return services.CreateOrchestrator(ctx).Run()
}
