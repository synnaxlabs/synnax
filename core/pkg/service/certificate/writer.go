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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, renames, and deletes certificates.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	table     *gorp.Table[Key, Certificate]
	now       func() telem.TimeStamp
	secrets   secret.Writer
	statuses  status.Writer
	group     group.Group
}

// Create creates the certificate, or replaces the one with the same key, and writes its
// expiry status. It returns a validation error for an invalid certificate or key.
//
// For a stored certificate, Create reads the details from the PEM and keeps only its
// certificate blocks. An optional PEM privateKey must match the leaf, and replaces the
// value of the certificate's key secret. Without one, a replacement keeps the old key
// only when the public key is unchanged. A file certificate takes no privateKey, and
// replacing a stored certificate with one deletes the old key secret.
func (w Writer) Create(ctx context.Context, c *Certificate, privateKey []byte) error {
	base := c.Base()
	if base.Key == uuid.Nil() {
		base.Key = uuid.New()
		*c = c.WithBase(base)
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
		v.Ternary(
			"private_key",
			len(privateKey) != 0,
			"must be empty for a file certificate",
		)
		if err = v.Error(); err != nil {
			return err
		}
		orphanedKey = prevStored.PrivateKey
	default:
		v := validate.New("certificate")
		v.Ternary("source", true, "required")
		return v.Error()
	}
	if err = w.table.NewCreate().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	if orphanedKey != nil {
		if err = w.secrets.Delete(ctx, *orphanedKey); err != nil {
			return err
		}
	}
	otgID := c.OntologyID()
	if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if err = w.otgWriter.DefineRelationships(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	); err != nil {
		return err
	}
	return w.writeStatus(ctx, *c)
}

func (w Writer) writeStatus(ctx context.Context, c Certificate) error {
	if st, ok := expiryStatus(c, w.now()); ok {
		return w.statuses.SetWithParent(ctx, &st, c.OntologyID())
	}
	return w.statuses.Delete(ctx, statusKey(c.Base().Key))
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
			Name: c.Name + " private key",
		}}}
		if err = w.secrets.Create(ctx, &key, privateKey); err != nil {
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
			"private_key",
			true,
			"required, because the new certificate does not match the saved key",
		)
		return StoredCertificate{}, v.Error()
	}
	return c, nil
}

// Rename renames the certificate with the given key and its expiry status. It returns
// query.ErrNotFound if no certificate has the key, and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	var renamed Certificate
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Certificate](key)).
		ChangeErr(func(_ gorp.Context, c Certificate) (Certificate, error) {
			base := c.Base()
			base.Name = name
			renamed = c.WithBase(base)
			if err := renamed.Validate(); err != nil {
				return Certificate{}, err
			}
			return renamed, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.writeStatus(ctx, renamed)
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
	if err := w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...); err != nil {
		return err
	}
	statusKeys := lo.Map(keys, func(k Key, _ int) status.Key { return statusKey(k) })
	if err := w.statuses.Delete(ctx, statusKeys...); err != nil {
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
