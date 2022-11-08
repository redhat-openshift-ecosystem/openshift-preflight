package cli

import (
	"context"
	"errors"
)

// badResultSubmitter implements ResultSubmitter and fails to submit with the included errmsg.
type badResultSubmitter struct {
	errmsg string
}

func (brs *badResultSubmitter) Submit(ctx context.Context) error {
	return errors.New(brs.errmsg)
}
