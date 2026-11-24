// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"context"
	"fmt"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Integration is the name that a rack lists in its integrations when it runs Kafka.
const Integration = "kafka"

// defaultPort is the port of a broker with no port.
const defaultPort uint16 = 9092

// validateFields validates c and refuses a cluster with no brokers, a broker or read
// with the natural key of another of its list.
func validateFields(c *Cluster) error {
	if err := c.Validate(); err != nil {
		return err
	}
	v := validate.New("Cluster")
	v.Ternary("brokers", len(c.Brokers) == 0, "must hold at least one broker")
	type address struct {
		host string
		port uint16
	}
	brokers := make(set.Set[address], len(c.Brokers))
	for i, b := range c.Brokers {
		k := address{host: b.Host, port: defaultPort}
		if b.Port != nil {
			k.port = *b.Port
		}
		v.Ternaryf(
			fmt.Sprintf("brokers.%d", i),
			brokers.Contains(k),
			"duplicates the broker %s:%d",
			k.host,
			k.port,
		)
		brokers.Add(k)
	}
	type read struct {
		topic, recordKey, pointer string
		message                   icd.MessageKey
	}
	reads := make(set.Set[read], len(c.Reads))
	for i, r := range c.Reads {
		k := read{
			topic:     r.Topic,
			recordKey: r.RecordKey,
			pointer:   r.Pointer,
			message:   r.Message,
		}
		v.Ternaryf(
			fmt.Sprintf("reads.%d", i),
			reads.Contains(k),
			"duplicates the read of %q from key %q of topic %s",
			r.Pointer,
			r.RecordKey,
			r.Topic,
		)
		reads.Add(k)
	}
	return v.Error()
}

// checkRack refuses a rack that does not run the Kafka integration.
func (w Writer) checkRack(ctx context.Context, c Cluster) error {
	var r rack.Rack
	if err := w.racks.NewRetrieve().
		Where(rack.MatchKeys(c.Rack)).
		Entry(&r).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if !slices.Contains(r.Integrations, Integration) {
		return errors.Wrapf(
			validate.ErrValidation,
			"rack: %s does not run the %s integration",
			r.Name,
			Integration,
		)
	}
	return nil
}
