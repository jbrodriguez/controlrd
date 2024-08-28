package api

import (
	"github.com/jbrodriguez/controlrd/daemon/dto"
	"github.com/jbrodriguez/controlrd/daemon/lib"
	"github.com/jbrodriguez/controlrd/daemon/logger"
)

func (a *Api) getOrigin() *dto.Origin {
	if a.origin == nil {
		origin, err := lib.GetOrigin()
		if err != nil {
			logger.Yellow(" unable to get origin: %s", err)
		}
		a.origin = origin
	}

	return a.origin
}
