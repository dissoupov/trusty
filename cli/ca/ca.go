package ca

import (
	"context"
	"fmt"

	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/trusty/cli"
	"github.com/go-phorce/trusty/pkg/print"
	"github.com/juju/errors"
)

// Issuers shows the Issuing CAs
func Issuers(c ctl.Control, _ interface{}) error {
	cli := c.(*cli.Cli)
	res, err := cli.Client().Authority.Issuers(context.Background())
	if err != nil {
		return errors.Trace(err)
	}

	if cli.IsJSON() {
		ctl.WriteJSON(c.Writer(), res)
		fmt.Fprint(c.Writer(), "\n")
	} else {
		print.Issuers(c.Writer(), res.Issuers, true)
	}
	return nil
}
