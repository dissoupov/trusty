package status

import (
	"context"
	"fmt"

	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/trusty/cli"
	"github.com/go-phorce/trusty/pkg/print"
	"github.com/juju/errors"
)

// Version shows the service version
func Version(c ctl.Control, _ interface{}) error {
	cli := c.(*cli.Cli)
	res, err := cli.Client().Status.Version(context.Background())
	if err != nil {
		return errors.Trace(err)
	}

	if cli.IsJSON() {
		ctl.WriteJSON(c.Writer(), res)
		fmt.Fprint(c.Writer(), "\n")
	} else {
		print.ServerVersion(c.Writer(), res)
	}
	return nil
}

// Server shows trusty server status
func Server(c ctl.Control, _ interface{}) error {
	cli := c.(*cli.Cli)
	res, err := cli.Client().Status.Server(context.Background())
	if err != nil {
		return errors.Trace(err)
	}

	if cli.IsJSON() {
		ctl.WriteJSON(c.Writer(), res)
		fmt.Fprint(c.Writer(), "\n")
	} else {
		print.ServerStatusResponse(c.Writer(), res)
	}

	return nil
}

// Caller shows the Caller status
func Caller(c ctl.Control, _ interface{}) error {
	cli := c.(*cli.Cli)
	res, err := cli.Client().Status.Caller(context.Background())
	if err != nil {
		return errors.Trace(err)
	}

	if cli.IsJSON() {
		ctl.WriteJSON(c.Writer(), res)
		fmt.Fprint(c.Writer(), "\n")
	} else {
		print.CallerStatusResponse(c.Writer(), res)
	}

	return nil
}
