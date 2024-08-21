package api

import (
	"github.com/jbrodriguez/controlrd/daemon/domain"
	"github.com/jbrodriguez/controlrd/daemon/dto"
)

type Api struct {
	ctx *domain.Context

	origin *dto.Origin

	// mailbox chan any
}

func Create(ctx *domain.Context) *Api {
	return &Api{
		ctx: ctx,
	}
}

func (a *Api) Run() error {
	return nil
}
