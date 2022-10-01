package fhttp

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type ClientConfig struct {
	EncoderDecoder httputil.EncoderDecoder
	Logger         *zap.SugaredLogger
}

func (c ClientConfig) Validate() error {
	v := validate.New("[ws.StreamClient]")
	validate.NotNil(v, "EncoderDecoder", c.EncoderDecoder)
	return v.Error()
}

func (c ClientConfig) Override(other ClientConfig) ClientConfig {
	c.EncoderDecoder = override.Nil(c.EncoderDecoder, other.EncoderDecoder)
	c.Logger = override.Nil(c.Logger, other.Logger)
	return c
}

var DefaultClientConfig = ClientConfig{
	EncoderDecoder: httputil.MsgPackEncoderDecoder,
	Logger:         zap.S(),
}

type Client struct {
	ClientConfig
}

func NewClient(configs ...ClientConfig) *Client {
	cfg, err := config.OverrideAndValidate(DefaultClientConfig, configs...)
	if err != nil {
		panic(err)
	}
	return &Client{ClientConfig: cfg}
}

func NewStreamClient[RQ, RS freighter.Payload](c *Client) freighter.StreamClient[RQ, RS] {
	return &StreamClient[RQ, RS]{
		logger: c.Logger,
		ecd:    c.EncoderDecoder,
	}
}
