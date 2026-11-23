// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package broker

import (
	"context"
	"fmt"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Integration is the name that a rack lists in its integrations when it runs MQTT.
const Integration = "mqtt"

type sparkplugTag struct{ group, edgeNode, device, tag string }

// validateFields validates b and refuses a keep alive of a fraction of a second, a
// Sparkplug group that cannot be a topic level, and a binding with the natural key of
// another binding of its list.
func validateFields(b *Broker) error {
	if err := b.Validate(); err != nil {
		return err
	}
	v := validate.New("Broker")
	v.Ternary(
		"keep_alive",
		b.KeepAlive%telem.Second != 0,
		"must be a whole number of seconds",
	)
	for i, g := range b.Sparkplug.Groups {
		v.Exec(func() error {
			return validate.PathedError(
				sparkplug.ValidateID("group", g),
				"sparkplug", "groups", fmt.Sprint(i),
			)
		})
	}
	type plainRead struct {
		topic   string
		message icd.MessageKey
		pointer string
	}
	plainReads := make(set.Set[plainRead], len(b.Reads))
	sparkplugReads := make(set.Set[sparkplugTag], len(b.Reads))
	for i, r := range b.Reads {
		path := fmt.Sprintf("reads.%d", i)
		switch r := r.Variant.(type) {
		case PlainRead:
			k := plainRead{topic: r.Topic, message: r.Message, pointer: r.Pointer}
			v.Ternaryf(
				path,
				plainReads.Contains(k),
				"duplicates the read of %q from topic %s",
				r.Pointer,
				r.Topic,
			)
			plainReads.Add(k)
		case SparkplugRead:
			k := sparkplugTag{r.Group, r.EdgeNode, r.Device, r.Tag}
			v.Ternaryf(path, sparkplugReads.Contains(k), "duplicates the read of tag %s", r.Tag)
			sparkplugReads.Add(k)
		}
	}
	type plainWrite struct {
		topic string
		field icd.FieldKey
	}
	plainWrites := make(set.Set[plainWrite], len(b.Writes))
	sparkplugWrites := make(set.Set[sparkplugTag], len(b.Writes))
	for i, w := range b.Writes {
		path := fmt.Sprintf("writes.%d", i)
		switch w := w.Variant.(type) {
		case PlainWrite:
			k := plainWrite{topic: w.Topic, field: w.Field}
			v.Ternaryf(path, plainWrites.Contains(k), "duplicates the write to topic %s", w.Topic)
			plainWrites.Add(k)
		case SparkplugWrite:
			k := sparkplugTag{w.Group, w.EdgeNode, w.Device, w.Tag}
			v.Ternaryf(
				path,
				sparkplugWrites.Contains(k),
				"duplicates the write to tag %s",
				w.Tag,
			)
			sparkplugWrites.Add(k)
		}
	}
	commands := make(set.Set[sparkplugTag], len(b.EdgeCommands))
	for i, c := range b.EdgeCommands {
		k := sparkplugTag{group: c.Group, edgeNode: c.EdgeNode, tag: c.Tag}
		v.Ternaryf(
			fmt.Sprintf("edge_commands.%d", i),
			commands.Contains(k),
			"duplicates the command of tag %s",
			c.Tag,
		)
		commands.Add(k)
	}
	return v.Error()
}

// checkRack refuses a rack that does not run the MQTT integration.
func (w Writer) checkRack(ctx context.Context, b Broker) error {
	var r rack.Rack
	if err := w.racks.NewRetrieve().
		Where(rack.MatchKeys(b.Rack)).
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

// checkClientCertificate refuses a client certificate that holds no private key.
func (w Writer) checkClientCertificate(ctx context.Context, b Broker) error {
	t, ok := b.Transport.Variant.(TlsTransport)
	if !ok || t.ClientCertificate == nil {
		return nil
	}
	var c certificate.Certificate
	if err := w.certs.NewRetrieve().
		Where(certificate.MatchKeys(*t.ClientCertificate)).
		Entry(&c).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	var hasKey bool
	switch v := c.Variant.(type) {
	case certificate.StoredCertificate:
		hasKey = v.PrivateKey != nil
	case certificate.FileCertificate:
		hasKey = v.PrivateKeyFile != nil
	}
	if !hasKey {
		return errors.Wrapf(
			validate.ErrValidation,
			"transport.client_certificate: certificate %q holds no private key",
			c.Base().Name,
		)
	}
	return nil
}
