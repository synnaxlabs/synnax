// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("v2 -> v3 Session migration", func() {
	It("Should stamp each session with its credential version", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		credential := uuid.New()
		kept := v2.Session{
			Key:        uuid.New(),
			Hash:       []byte{1, 2},
			Credential: credential,
			LastActive: 5,
		}
		orphan := v2.Session{Key: uuid.New(), Credential: uuid.New()}
		Expect(gorp.WrapWriter[v2.SessionKey, v2.Session](db).
			Set(ctx, kept, orphan)).To(Succeed())
		version := func(
			_ context.Context,
			_ gorp.Tx,
			key uuid.UUID,
		) (telem.TimeStamp, error) {
			if key == credential {
				return 7, nil
			}
			return 0, errors.Wrap(query.ErrNotFound, "credential")
		}
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Session",
			Migrations: []gorp.Migration{v3.NewMigration(version)},
		})).To(Succeed())
		var sessions []v3.Session
		Expect(gorp.NewRetrieve[v3.SessionKey, v3.Session]().
			Entries(&sessions).
			Exec(ctx, db)).To(Succeed())
		Expect(sessions).To(ConsistOf(v3.Session{
			Key:               kept.Key,
			Hash:              kept.Hash,
			Credential:        credential,
			CredentialVersion: 7,
			LastActive:        5,
		}))
	})
})
