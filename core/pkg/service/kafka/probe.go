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
	"github.com/synnaxlabs/x/errors"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// stageTimeout bounds one stage of a connection test.
const stageTimeout = 5 * time.Second

// defaultPort is the port of a seed broker whose address names none.
const defaultPort = "9092"

// testConnection tests a connection to the cluster of props one stage at a time: dial,
// TLS, SASL, metadata, and each broker that the cluster advertises. It returns a
// driver.Failure for the first stage that fails, and a validation error when props
// cannot open a client.
func testConnection(ctx context.Context, props Properties) error {
	props.ApplyDefaults()
	if err := validateProperties(props); err != nil {
		return err
	}
	conn, addr, err := dialSeed(ctx, props.Brokers)
	if err != nil {
		return err
	}
	if props.Secure {
		err = handshake(ctx, conn, addr, props)
	}
	if cErr := conn.Close(); cErr != nil && !errors.Is(cErr, net.ErrClosed) {
		return cErr
	}
	if err != nil {
		return err
	}
	return testCluster(ctx, props)
}

// dialSeed opens a TCP connection to the first seed broker that answers, and returns
// its address.
func dialSeed(ctx context.Context, brokers []string) (net.Conn, string, error) {
	var (
		dialer   net.Dialer
		failures []string
	)
	for _, addr := range brokers {
		addr = withPort(addr)
		dialCtx, cancel := context.WithTimeout(ctx, stageTimeout)
		conn, err := dialer.DialContext(dialCtx, "tcp", addr)
		cancel()
		if err == nil {
			return conn, addr, nil
		}
		failures = append(failures, fmt.Sprintf("Could not reach %s: %s.", addr, err))
	}
	return nil, "", driver.Failure{
		Message: "Failed to reach broker",
		Description: strings.Join(failures, " ") + " Check the address and port of " +
			"each broker, and that the network of the Core can reach them.",
	}
}

// withPort returns addr with defaultPort when it names no port, as the Kafka client
// reads a seed broker address.
func withPort(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), defaultPort)
}

// handshake runs the TLS handshake of props on conn, the connection to addr.
func handshake(
	ctx context.Context,
	conn net.Conn,
	addr string,
	props Properties,
) error {
	ctx, cancel := context.WithTimeout(ctx, stageTimeout)
	defer cancel()
	return driver.Handshake(ctx, conn, addr, tlsConfig(props))
}

// testCluster authenticates, reads the cluster metadata, and reaches each broker that
// the metadata advertises.
func testCluster(ctx context.Context, props Properties) error {
	var auth authHook
	cl, err := openClient(
		props,
		kgo.DialTimeout(stageTimeout),
		kgo.RequestRetries(1),
		kgo.WithHooks(&auth),
	)
	if err != nil {
		return driver.Failure{
			Message:     "Failed to open client",
			Description: err.Error() + ".",
		}
	}
	defer cl.Close()
	mdCtx, cancel := context.WithTimeout(ctx, stageTimeout)
	md, err := kadm.NewClient(cl).BrokerMetadata(mdCtx)
	cancel()
	if err == nil {
		return testBrokers(ctx, cl, md.Brokers)
	}
	if props.Sasl.Mechanism() != NoneSASLType &&
		(saslFailure(err) || auth.dropped.Load()) {
		return driver.Failure{
			Message: "Authentication failed",
			Description: fmt.Sprintf(
				"%s. Check the username, the password, and the SASL mechanism.", err,
			),
		}
	}
	return metadataFailure(err)
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

func metadataFailure(err error) error {
	const hint = "Check that the TLS and SASL settings match the listener of the broker."
	switch {
	case errors.Is(err, kerr.ClusterAuthorizationFailed),
		errors.Is(err, kerr.TopicAuthorizationFailed):
		return driver.Failure{
			Message: "Not authorized to read metadata",
			Description: fmt.Sprintf(
				"%s. Grant the user the Describe operation on the cluster.", err,
			),
		}
	case errors.Is(err, context.DeadlineExceeded):
		return driver.Failure{
			Message: "Broker did not answer",
			Description: fmt.Sprintf(
				"The broker did not answer within %s. %s", stageTimeout, hint,
			),
		}
	default:
		return driver.Failure{
			Message:     "Failed to read cluster metadata",
			Description: fmt.Sprintf("%s. %s", err, hint),
		}
	}
}

// testBrokers reaches each broker of brokers through cl, which dials the address that
// the cluster advertises for it. It returns a driver.Failure for the broker with the
// lowest node ID that it cannot reach.
func testBrokers(
	ctx context.Context,
	cl *kgo.Client,
	brokers kadm.BrokerDetails,
) error {
	brokers = slices.Clone(brokers)
	slices.SortFunc(brokers, func(a, b kadm.BrokerDetail) int {
		return int(a.NodeID - b.NodeID)
	})
	failures := make([]error, len(brokers))
	var wg sync.WaitGroup
	for i, b := range brokers {
		wg.Go(func() {
			addr := net.JoinHostPort(b.Host, strconv.Itoa(int(b.Port)))
			ctx, cancel := context.WithTimeout(ctx, stageTimeout)
			defer cancel()
			req := kmsg.NewPtrApiVersionsRequest()
			if _, err := cl.Broker(int(b.NodeID)).Request(ctx, req); err != nil {
				failures[i] = driver.Failure{
					Message: "Failed to reach broker",
					Description: fmt.Sprintf(
						"The cluster advertises broker %d at %s, which the Core cannot "+
							"reach: %s. Set the advertised listeners of the broker to an "+
							"address that the Core can reach.",
						b.NodeID,
						addr,
						err,
					),
				}
			}
		})
	}
	wg.Wait()
	for _, err := range failures {
		if err != nil {
			return err
		}
	}
	return nil
}
