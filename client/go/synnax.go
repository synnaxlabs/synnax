//go:generate go run ./cmd/gen/main.go ../../x/go/telem telem.go
package synnax

import (
	"github.com/synnaxlabs/client/auth"
	"github.com/synnaxlabs/client/channel"
	"github.com/synnaxlabs/client/framer"
	"github.com/synnaxlabs/client/transport/grpc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Synnax is the client library for interacting with a Synnax cluster. It should not
// be instantiated directly, but instead constructed using the synnax.New method.
type Synnax struct {
	*framer.Client
	// Channels is the client for creating, retrieving, and deleting channels within
	// a Synnax cluster.
	Channels *channel.Client
}

// Config is the configuration for a Synnax client. It should be passed to the New
// method to create a new Synnax client.
type Config struct {
	// Host is the host IP address of a node in the Synnax cluster.
	Host string
	// Port is the port of the node at the given Host.
	Port int
	// Username is the username to authenticate with the Synnax cluster.
	Username string
	// Password is the password to authenticate with the Synnax cluster.
	Password string
	// Secure sets whether to use TLS when communicating with the Synnax cluster. This
	// should be set to false for clusters started with the --insecure flag.
	Secure bool
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a Synnax client. It is not
	// sufficient on its own, and should be overridden with the desired configuration.
	DefaultConfig = Config{Secure: false}
)

// Validate implements config.Config. Internal use only.
func (c Config) Validate() error {
	v := validate.New("Synnax.Config")
	validate.NotEmptyString(v, "Host", c.Host)
	validate.Positive(v, "Port", c.Port)
	validate.NotEmptyString(v, "Username", c.Username)
	validate.NotEmptyString(v, "Password", c.Password)
	return v.Error()
}

// Override implements config.Config. Internal use only.
func (c Config) Override(other Config) Config {
	c.Host = override.String(c.Host, other.Host)
	c.Port = override.Numeric(c.Port, other.Port)
	c.Username = override.String(c.Username, other.Username)
	c.Password = override.String(c.Password, other.Password)
	return c
}

// Open creates a new Synnax client with the given configuration, returning an error
// if the configuration is invalid.
func Open(cfgs ...Config) (*Synnax, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	transport := grpc.New(address.Newf("%s:%v", cfg.Host, cfg.Port))
	a := auth.New(transport.AuthLogin, auth.InsecureCredentials{
		Username: cfg.Username,
		Password: password.Raw(cfg.Password),
	})
	transport.Use(a.Middleware())
	channelClient := channel.NewClient(
		transport.ChannelCreate,
		transport.ChannelRetrieve,
		transport.ChannelDelete,
	)
	return &Synnax{
		Channels: channelClient,
		Client: framer.NewClient(
			channelClient,
			transport.FrameIterator,
			transport.FrameWriter,
			transport.FrameStreamer,
		),
	}, nil
}
