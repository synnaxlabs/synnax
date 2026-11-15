// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, changes, and deletes certificates.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Certificate]
	now        func() telem.TimeStamp
	secrets    secret.Writer
	statuses   status.Writer
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Set creates the certificate, or replaces the one with the same key, and writes its
// expiry status. It returns a validation error for an invalid certificate or key.
//
// For a stored certificate, Set reads the details from the PEM and keeps only its
// certificate blocks. An optional PEM privateKey must match the leaf, and replaces the
// value of the certificate's key secret. Without one, a replacement keeps the old key
// only when the public key is unchanged, and renames its secret after a new name. A
// file certificate takes no privateKey, and replacing a stored certificate with one
// deletes the old key secret. The commit carries a create action that holds the
// certificate.
func (w Writer) Set(ctx context.Context, c *Certificate, privateKey []byte) error {
	if c.Variant == nil {
		v := validate.New("certificate")
		v.Ternary("source", true, "required")
		return v.Error()
	}
	base := c.Base()
	if base.Key == uuid.Nil() {
		base.Key = uuid.New()
		c.SetBase(base)
	}
	if err := c.Validate(); err != nil {
		return err
	}
	var prev Certificate
	err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Certificate](base.Key)).
		Entry(&prev).
		Exec(ctx, w.tx)
	if errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}
	prevStored, _ := prev.Variant.(StoredCertificate)
	var orphanedKey *secret.Key
	switch variant := c.Variant.(type) {
	case StoredCertificate:
		if variant, err = w.prepareStored(
			ctx,
			variant,
			privateKey,
			prevStored,
		); err != nil {
			return err
		}
		c.Variant = variant
	case FileCertificate:
		v := validate.New("certificate")
		v.NotEmptyString("certificate_path", variant.CertificatePath)
		if variant.PrivateKeyFile != nil {
			secret.ValidateFileName(v, "private_key_file", *variant.PrivateKeyFile)
		}
		v.Ternary(
			"private_key_pem",
			len(privateKey) != 0,
			"must be empty for a file certificate",
		)
		if err = v.Error(); err != nil {
			return err
		}
		orphanedKey = prevStored.PrivateKey
	}
	if err = w.table.NewCreate().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, base.Key, NewCreateAction(CreatePayload{Certificate: *c}))
	if orphanedKey != nil {
		if err = w.secrets.Delete(ctx, *orphanedKey); err != nil {
			return err
		}
	}
	return w.writeStatus(ctx, *c)
}

func (w Writer) writeStatus(ctx context.Context, c Certificate) error {
	if st, ok := expiryStatus(c, w.now()); ok {
		return w.statuses.Set(ctx, &st)
	}
	return w.statuses.DeleteOwned(ctx, c.ResourceID())
}

func (w Writer) prepareStored(
	ctx context.Context,
	c StoredCertificate,
	privateKey []byte,
	prev StoredCertificate,
) (StoredCertificate, error) {
	chain, leaf, err := parseChain(c.Pem)
	if err != nil {
		return StoredCertificate{}, err
	}
	c.Pem = chain
	c.Details = newDetails(leaf)
	c.PrivateKey = prev.PrivateKey
	if len(privateKey) != 0 {
		if err = checkPrivateKey(leaf, privateKey); err != nil {
			return StoredCertificate{}, err
		}
		key := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{
			Key:  lo.FromPtr(c.PrivateKey),
			Name: keyName(c.Name),
		}}}
		if err = w.secrets.Set(ctx, &key, privateKey); err != nil {
			return StoredCertificate{}, err
		}
		c.PrivateKey = new(key.Base().Key)
		return c, nil
	}
	if c.PrivateKey == nil {
		return c, nil
	}
	_, prevLeaf, err := parseChain(prev.Pem)
	if err != nil {
		return StoredCertificate{}, err
	}
	if !samePublicKey(prevLeaf.PublicKey, leaf.PublicKey) {
		v := validate.New("certificate")
		v.Ternary(
			"private_key_pem",
			true,
			"required, because the new certificate does not match the saved key",
		)
		return StoredCertificate{}, v.Error()
	}
	if err = w.renameKey(ctx, c, prev.Name); err != nil {
		return StoredCertificate{}, err
	}
	return c, nil
}

// keyName is the name of the secret that holds the private key of the certificate with
// the given name.
func keyName(certificate string) string { return certificate + " private key" }

func (w Writer) renameKey(ctx context.Context, c StoredCertificate, prev string) error {
	if c.PrivateKey == nil || c.Name == prev {
		return nil
	}
	return w.secrets.Rename(ctx, *c.PrivateKey, keyName(c.Name))
}

// Apply applies the actions to the certificate with the given key, renames its key
// secret after a new name, rewrites its expiry status, and attaches the actions to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create or the
// result fails to validate, and an error wrapping query.ErrNotFound when no certificate
// has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var prev, next Certificate
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Certificate](key)).
		ChangeErr(func(_ gorp.Context, c Certificate) (Certificate, error) {
			var err error
			prev = c
			if next, err = Reduce(c, acts...); err != nil {
				return Certificate{}, err
			}
			if err = next.Validate(); err != nil {
				return Certificate{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if stored, ok := next.Variant.(StoredCertificate); ok {
		if err := w.renameKey(ctx, stored, prev.Base().Name); err != nil {
			return err
		}
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return w.writeStatus(ctx, next)
}

// Rename renames the certificate with the given key, its key secret, and its expiry
// status by applying a set of the name.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

// Delete deletes the certificates with the given keys, the secrets that hold their
// private keys, and their expiry statuses. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	var certs []Certificate
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Certificate](keys...)).
		Entries(&certs).
		Exec(ctx, w.tx); errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Certificate](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	privateKeys := lo.FilterMap(certs, func(c Certificate, _ int) (secret.Key, bool) {
		key := privateKey(c)
		return lo.FromPtr(key), key != nil
	})
	if len(privateKeys) == 0 {
		return nil
	}
	return w.secrets.Delete(ctx, privateKeys...)
}

func privateKey(c Certificate) *secret.Key {
	if stored, ok := c.Variant.(StoredCertificate); ok {
		return stored.PrivateKey
	}
	return nil
}
