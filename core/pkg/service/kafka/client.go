// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"
	"crypto/tls"

	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Make is the device make of a Kafka cluster.
const Make = "kafka"

// decodeProperties parses the connection properties of a Kafka cluster device. It
// returns an error wrapping validate.ErrValidation when dev is not a Kafka cluster or
// its properties are incomplete.
func decodeProperties(dev device.Device) (Properties, error) {
	var props Properties
	if dev.Make != Make {
		return props, errors.Wrapf(
			validate.ErrValidation, "device %s is not a Kafka cluster", dev.Key,
		)
	}
	if err := dev.Properties.Unmarshal(&props); err != nil {
		return props, errors.Wrapf(err, "parsing properties of device %s", dev.Key)
	}
	props.ApplyDefaults()
	return props, validateProperties(props)
}

// validateProperties checks that props can open a client.
func validateProperties(props Properties) error {
	v := validate.New("kafka.properties")
	v.NotEmptySlice("brokers", props.Brokers)
	for i, b := range props.Brokers {
		v.Ternaryf("brokers", b == "", "broker %d must not be empty", i)
	}
	return v.Error()
}

// clientOpts translates props into franz-go client options.
func clientOpts(props Properties) ([]kgo.Opt, error) {
	if err := validateProperties(props); err != nil {
		return nil, err
	}
	opts := []kgo.Opt{kgo.SeedBrokers(props.Brokers...)}
	if props.TLS {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		opts = append(opts, kgo.DialTLSConfig(tlsCfg))
	}
	switch v := props.Sasl.Variant.(type) {
	case nil, NoneSASL:
	case PlainSASL:
		auth := plain.Auth{User: v.Username, Pass: v.Password}
		opts = append(opts, kgo.SASL(auth.AsMechanism()))
	case ScramSha256SASL:
		auth := scram.Auth{User: v.Username, Pass: v.Password}
		opts = append(opts, kgo.SASL(auth.AsSha256Mechanism()))
	case ScramSha512SASL:
		auth := scram.Auth{User: v.Username, Pass: v.Password}
		opts = append(opts, kgo.SASL(auth.AsSha512Mechanism()))
	default:
		return nil, errors.Newf("unknown SASL mechanism %T", v)
	}
	return opts, nil
}

// openClient opens a client to the cluster in props. extra are appended to the
// connection options, so a consumer or producer can add its own.
func openClient(props Properties, extra ...kgo.Opt) (*kgo.Client, error) {
	opts, err := clientOpts(props)
	if err != nil {
		return nil, err
	}
	return kgo.NewClient(append(opts, extra...)...)
}

// ping opens a client to the cluster in props and checks that a broker answers
// before ctx ends.
func ping(ctx context.Context, props Properties) error {
	cl, err := openClient(props)
	if err != nil {
		return err
	}
	defer cl.Close()
	return cl.Ping(ctx)
}
