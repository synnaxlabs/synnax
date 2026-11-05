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
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// stageTimeout bounds one stage of a connection test.
const stageTimeout = 5 * time.Second

// The stages of a connection test.
const (
	stageDial     = "dial"
	stageSASL     = "sasl"
	stageMetadata = "metadata"
	stageBroker   = "broker"
)

// testConnection tests a connection to the cluster of props one stage at a time: dial,
// TLS, SASL, metadata, and each broker that the cluster advertises. It stops after the
// first stage that fails. It returns a validation error when props cannot open a
// client.
func testConnection(ctx context.Context, props Properties) ([]task.TestStep, error) {
	props.ApplyDefaults()
	if err := validateProperties(props); err != nil {
		return nil, err
	}
	conn, step := dialSeed(ctx, props.Brokers)
	steps := []task.TestStep{step}
	if conn == nil {
		return steps, nil
	}
	if props.TLS {
		step = handshake(ctx, conn, step.Address, props)
		steps = append(steps, step)
	}
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return nil, err
	}
	if step.Variant == status.VariantError {
		return steps, nil
	}
	return append(steps, testCluster(ctx, props)...), nil
}

func success(stage, address, message string) task.TestStep {
	return task.TestStep{
		Stage:   stage,
		Address: address,
		Variant: status.VariantSuccess,
		Message: message,
	}
}

func failure(stage, address, message, description string) task.TestStep {
	return task.TestStep{
		Stage:       stage,
		Address:     address,
		Variant:     status.VariantError,
		Message:     message,
		Description: description,
	}
}

// dialSeed opens a TCP connection to the first seed broker that answers. It returns a
// nil connection when none answers.
func dialSeed(ctx context.Context, brokers []string) (net.Conn, task.TestStep) {
	var (
		dialer   net.Dialer
		failures []string
	)
	for _, addr := range brokers {
		dialCtx, cancel := context.WithTimeout(ctx, stageTimeout)
		conn, err := dialer.DialContext(dialCtx, "tcp", addr)
		cancel()
		if err != nil {
			failures = append(
				failures,
				fmt.Sprintf("Could not reach %s: %s.", addr, err),
			)
			continue
		}
		step := success(stageDial, addr, "Reached broker")
		if len(failures) > 0 {
			step.Variant = status.VariantWarning
			step.Message = "Some seed brokers not reachable"
			step.Description = strings.Join(failures, " ")
		}
		return conn, step
	}
	return nil, failure(
		stageDial,
		"",
		"Failed to reach broker",
		strings.Join(failures, " ")+" Check the address and port of each broker, and "+
			"that the network of the Core can reach them.",
	)
}

// handshake runs the TLS handshake of props on conn, the connection to addr.
func handshake(
	ctx context.Context,
	conn net.Conn,
	addr string,
	props Properties,
) task.TestStep {
	ctx, cancel := context.WithTimeout(ctx, stageTimeout)
	defer cancel()
	return driver.Handshake(ctx, conn, addr, tlsConfig(props))
}

// testCluster authenticates, reads the cluster metadata, and reaches each broker that
// the metadata advertises.
func testCluster(ctx context.Context, props Properties) []task.TestStep {
	var auth authHook
	cl, err := openClient(
		props,
		kgo.DialTimeout(stageTimeout),
		kgo.RequestRetries(1),
		kgo.WithHooks(&auth),
	)
	if err != nil {
		return []task.TestStep{
			failure(stageMetadata, "", "Failed to open client", err.Error()+"."),
		}
	}
	defer cl.Close()
	mdCtx, cancel := context.WithTimeout(ctx, stageTimeout)
	md, err := kadm.NewClient(cl).BrokerMetadata(mdCtx)
	cancel()
	sasl := props.Sasl.Mechanism() != NoneSASLType
	if err != nil {
		if sasl && (saslFailure(err) || auth.dropped.Load()) {
			return []task.TestStep{failure(
				stageSASL, "", "Authentication failed", fmt.Sprintf(
					"%s. Check the username, the password, and the SASL mechanism.",
					err,
				),
			)}
		}
		return []task.TestStep{metadataFailure(err)}
	}
	var steps []task.TestStep
	if sasl {
		steps = append(steps, success(stageSASL, "", "Authenticated"))
	}
	steps = append(steps, success(stageMetadata, "", "Read cluster metadata"))
	return append(steps, testBrokers(ctx, cl, md.Brokers)...)
}

// authHook records whether a broker closed a connection while it authenticated the
// client, which some brokers do in place of an error response.
type authHook struct{ dropped atomic.Bool }

var _ kgo.HookBrokerE2E = (*authHook)(nil)

// OnBrokerE2E implements kgo.HookBrokerE2E.
func (h *authHook) OnBrokerE2E(_ kgo.BrokerMetadata, key int16, e2e kgo.BrokerE2E) {
	if key == kmsg.SASLAuthenticate.Int16() && e2e.WriteErr == nil &&
		e2e.ReadErr != nil {
		h.dropped.Store(true)
	}
}

func saslFailure(err error) bool {
	return errors.Is(err, kerr.SaslAuthenticationFailed) ||
		errors.Is(err, kerr.UnsupportedSaslMechanism) ||
		errors.Is(err, kerr.IllegalSaslState)
}

func metadataFailure(err error) task.TestStep {
	const hint = "Check that the TLS and SASL settings match the listener of the broker."
	switch {
	case errors.Is(err, kerr.ClusterAuthorizationFailed),
		errors.Is(err, kerr.TopicAuthorizationFailed):
		return failure(
			stageMetadata,
			"",
			"Not authorized to read metadata",
			fmt.Sprintf(
				"%s. Grant the user the Describe operation on the cluster.", err,
			),
		)
	case errors.Is(err, context.DeadlineExceeded):
		return failure(stageMetadata, "", "Broker did not answer", fmt.Sprintf(
			"The broker did not answer within %s. %s", stageTimeout, hint,
		))
	default:
		return failure(
			stageMetadata,
			"",
			"Failed to read cluster metadata",
			fmt.Sprintf(
				"%s. %s", err, hint,
			),
		)
	}
}

// testBrokers reaches each broker of brokers through cl, which dials the address that
// the cluster advertises for it.
func testBrokers(
	ctx context.Context,
	cl *kgo.Client,
	brokers kadm.BrokerDetails,
) []task.TestStep {
	brokers = slices.Clone(brokers)
	slices.SortFunc(brokers, func(a, b kadm.BrokerDetail) int {
		return int(a.NodeID - b.NodeID)
	})
	steps := make([]task.TestStep, len(brokers))
	var wg sync.WaitGroup
	for i, b := range brokers {
		wg.Go(func() {
			addr := net.JoinHostPort(b.Host, strconv.Itoa(int(b.Port)))
			ctx, cancel := context.WithTimeout(ctx, stageTimeout)
			defer cancel()
			req := kmsg.NewPtrApiVersionsRequest()
			if _, err := cl.Broker(int(b.NodeID)).Request(ctx, req); err != nil {
				steps[i] = failure(
					stageBroker,
					addr,
					"Failed to reach broker",
					fmt.Sprintf(
						"The cluster advertises broker %d at %s, which the Core cannot "+
							"reach: %s. Set the advertised listeners of the broker to an "+
							"address that the Core can reach.",
						b.NodeID,
						addr,
						err,
					),
				)
				return
			}
			steps[i] = success(stageBroker, addr, "Reached broker")
		})
	}
	wg.Wait()
	return steps
}
