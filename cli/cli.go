// Package cli provides common code for building a command line control for the service
package cli

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/trusty/client"
	"github.com/go-phorce/trusty/config"
	"github.com/go-phorce/trusty/pkg/inmemcrypto"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/trusty", "cli")

// ReturnCode is the type that your command returns, these map to standard process return codes
type ReturnCode ctl.ReturnCode

// A Option modifies the default behavior of Client.
type Option interface {
	applyOption(*Cli)
}

type optionFunc func(*Cli)

func (f optionFunc) applyOption(opts *Cli) { f(opts) }

type cliOptions struct {
}

// WithServer specifies to enable --server, --retry, --timeout flags
//
//
func WithServer(defaultServerURL string) Option {
	return optionFunc(func(c *Cli) {
		app := c.App()
		serverFlag := app.Flag("server", "URL of the server to control").Short('s')
		if defaultServerURL != "" {
			serverFlag = serverFlag.Default(defaultServerURL)
		}
		c.flags.server = serverFlag.String()
		c.flags.retries = app.Flag("retries", "number of retries for connect failures").Default("0").Int()
		c.flags.timeout = app.Flag("timeout", "timeout in seconds").Default("6").Int()
		c.flags.ctJSON = app.Flag("json", "print responses as JSON").Bool()
	})
}

// WithTLS specifies to enable --tls flags
func WithTLS() Option {
	return optionFunc(func(c *Cli) {
		app := c.App()
		c.flags.certFile = app.Flag("tls-cert", "client certificate for TLS connection").Short('c').String()
		c.flags.keyFile = app.Flag("tls-key", "key file for client certificate").Short('k').String()
		c.flags.trustedCAFile = app.Flag("tls-trusted-ca", "trusted CA certificate file for TLS connection").Short('r').Default("").String()
	})
}

// WithServiceCfg specifies to enable --cfg flag
func WithServiceCfg() Option {
	return optionFunc(func(c *Cli) {
		c.flags.serviceConfig = c.App().Flag("cfg", "trusty configuration file").Default(config.ConfigFileName).String()
	})
}

// WithHsmCfg specifies to enable --hsm-cfg flag
func WithHsmCfg() Option {
	return optionFunc(func(c *Cli) {
		app := c.App()
		c.flags.hsmConfig = app.Flag("hsm-cfg", "HSM provider configuration file").String()
		c.flags.plainKey = app.Flag("plain-key", "generate plain-text key, not in HSM").Bool()
	})
}

// ReadFileOrStdinFn allows to read from file or Stdin if the name is "-"
type ReadFileOrStdinFn func(filename string) ([]byte, error)

// Cli is a trusty specific wrapper to the ctl.Cli struct
type Cli struct {
	*ctl.Ctl
	ReadFileOrStdin ReadFileOrStdinFn

	flags struct {
		debug   *bool
		verbose *bool
		// server URLs
		server *string
		// serviceConfig specifies service configuration file
		serviceConfig *string
		// hsmConfig specifies HSM configuration file
		hsmConfig *string
		plainKey  *bool

		certFile      *string
		keyFile       *string
		trustedCAFile *string

		// ctJSON specifies to print responses as JSON
		ctJSON *bool
		// Retry settings
		retries *int
		timeout *int
	}

	config            *config.Configuration
	crypto            *cryptoprov.Crypto
	defaultCryptoProv cryptoprov.Provider
	client            *client.Client
}

// New creates an instance of trusty CLI
func New(d *ctl.ControlDefinition, opts ...Option) *Cli {
	cli := &Cli{
		ReadFileOrStdin: ReadStdin,
		Ctl:             ctl.NewControl(d),
	}

	cli.flags.verbose = d.App.Flag("verbose", "verbose output").Short('V').Bool()
	cli.flags.debug = d.App.Flag("debug", "redirect logs to stderr").Short('D').Bool()

	for _, opt := range opts {
		opt.applyOption(cli)
	}

	return cli
}

// Close allocated resources
func (cli *Cli) Close() {
	if cli.client != nil {
		cli.client.Close()
		cli.client = nil
	}
}

// Verbose specifies if verbose output is enabled
func (cli *Cli) Verbose() bool {
	return cli.flags.verbose != nil && *cli.flags.verbose
}

// IsJSON specifies if JSON output is required
func (cli *Cli) IsJSON() bool {
	return cli.flags.ctJSON != nil && *cli.flags.ctJSON
}

// ConfigFlag returns --cfg flag
func (cli *Cli) ConfigFlag() string {
	return *cli.flags.serviceConfig
}

// Config returns service configuration
func (cli *Cli) Config() *config.Configuration {
	if cli == nil || cli.config == nil {
		panic("use EnsureServiceConfig() in App settings")
	}
	return cli.config
}

// CryptoProv returns crypto provider
func (cli *Cli) CryptoProv() (*cryptoprov.Crypto, cryptoprov.Provider) {
	if cli == nil || cli.crypto == nil {
		panic("use EnsureCryptoProvider() in App settings")
	}
	return cli.crypto, cli.defaultCryptoProv
}

// Server returns server URL
func (cli *Cli) Server() string {
	if cli.flags.server != nil {
		return *cli.flags.server
	}
	return ""
}

// RegisterAction create new Control action
func (cli *Cli) RegisterAction(f func(c ctl.Control, flags interface{}) error, params interface{}) ctl.Action {
	return func() error {
		err := f(cli, params)
		if err != nil {
			return cli.Fail("action failed", err)
		}
		return nil
	}
}

// EnsureServiceConfig is pre-action to load service configuration
func (cli *Cli) EnsureServiceConfig() error {
	if cli.config == nil && cli.flags.serviceConfig != nil && *cli.flags.serviceConfig != "" {
		var err error
		cli.config, err = config.LoadConfig(*cli.flags.serviceConfig)
		if err != nil {
			return errors.Annotate(err, "load service configuration")
		}
	}
	if cli.config == nil {
		return errors.Errorf("specify --cfg option")
	}
	return nil
}

// EnsureCryptoProvider is pre-action to load Crypto provider
func (cli *Cli) EnsureCryptoProvider() error {
	if cli.crypto != nil {
		return nil
	}

	var err error
	var defaultProvider string
	var providers []string

	if cli.flags.hsmConfig != nil && *cli.flags.hsmConfig != "" {
		defaultProvider = *cli.flags.hsmConfig
	} else if cli.flags.serviceConfig != nil && *cli.flags.serviceConfig != "" {
		err = cli.EnsureServiceConfig()
		if err != nil {
			return errors.Trace(err)
		}
		defaultProvider = cli.config.CryptoProv.Default
		providers = cli.config.CryptoProv.Providers
	}

	cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)
	cryptoprov.Register("PKCS11", cryptoprov.Crypto11Loader)
	cli.crypto, err = cryptoprov.Load(defaultProvider, providers)
	if err != nil {
		return errors.Annotate(err, "unable to initialize crypto providers")
	}

	if cli.flags.plainKey != nil && *cli.flags.plainKey == true {
		cli.defaultCryptoProv = inmemcrypto.NewProvider()
	} else {
		cli.defaultCryptoProv = cli.crypto.Default()
	}

	return nil
}

// WithCryptoProvider sets custom Crypto Provider
func (cli *Cli) WithCryptoProvider(crypto *cryptoprov.Crypto) {
	cli.crypto = crypto
}

// PopulateControl is a pre-action for kingpin library to populate the
// control object after all the flags are parsed
func (cli *Cli) PopulateControl() error {
	isDebug := *cli.flags.debug
	var sink io.Writer
	if isDebug {
		sink = os.Stderr
		xlog.SetFormatter(xlog.NewColorFormatter(sink, true))
		xlog.SetGlobalLogLevel(xlog.DEBUG)
	} else {
		xlog.SetGlobalLogLevel(xlog.CRITICAL)
	}
	return nil
}

// WithClient sets gRPC client
func (cli *Cli) WithClient(c *client.Client) *Cli {
	if cli.client != nil {
		cli.client.Close()
		cli.client = nil
	}
	cli.client = c
	return cli
}

// Client returns client instance with active gRPC connection
func (cli *Cli) Client() *client.Client {
	if cli == nil || cli.client == nil {
		panic("use EnsureClient() in App settings")
	}
	return cli.client
}

// EnsureClient is pre-action to instantiate trusty client
func (cli *Cli) EnsureClient() error {
	if cli.client != nil {
		return nil
	}

	var tlsCert, tlsKey, tlsCA string
	if cli.flags.certFile != nil && *cli.flags.certFile != "" {
		tlsCert = *cli.flags.certFile
		tlsKey = *cli.flags.keyFile
		tlsCA = *cli.flags.trustedCAFile
	}

	var cfg *config.Configuration

	if cli.flags.serviceConfig != nil && *cli.flags.serviceConfig != "" {
		err := cli.EnsureServiceConfig()
		if err != nil {
			return errors.Trace(err)
		}

		cfg = cli.Config()
		if cli.Server() == "" && len(cfg.TrustyClient.Servers) > 0 {
			*cli.flags.server = cfg.TrustyClient.Servers[0]
		}
	}

	if cli.Server() == "" {
		return errors.New("use --server option")
	}

	if (tlsCert == "" || tlsKey == "") && cfg != nil {
		tlsCert = cfg.TrustyClient.ClientTLS.CertFile
		tlsKey = cfg.TrustyClient.ClientTLS.KeyFile
		tlsCA = cfg.TrustyClient.ClientTLS.TrustedCAFile
	}

	tlscfg, err := tlsconfig.NewClientTLSFromFiles(
		tlsCert,
		tlsKey,
		tlsCA)
	if err != nil {
		return errors.Annotate(err, "unable to build TLS configuration")
	}

	// TODO: add timeout options

	clientCfg := &client.Config{
		Endpoints: []string{cli.Server()},
		TLS:       tlscfg,
	}

	cli.client, err = client.New(clientCfg)
	if err != nil {
		return errors.Annotate(err, "unable to create client")
	}
	return nil
}

// ReadStdin reads from stdin if the file is "-"
func ReadStdin(filename string) ([]byte, error) {
	if filename == "" {
		return nil, errors.New("empty file name")
	}
	if filename == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(filename)
}
