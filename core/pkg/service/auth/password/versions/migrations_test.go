// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password/versions"
	authversions "github.com/synnaxlabs/synnax/pkg/service/auth/versions"
)

var _ = Describe("Legacy migrations", func() {
	It("Should run under the credentials table's namespace", func() {
		Expect(versions.LegacyNamespace).To(Equal(authversions.Namespace))
		Expect(versions.LegacyNamespace).To(Equal("SecureCredentials"))
	})
	It("Should lift the credentials table before moving it", func() {
		migrations := versions.NewLegacyMigrations(versions.LegacyConfig{})
		Expect(migrations).To(HaveLen(len(authversions.Migrations) + 1))
		Expect(migrations[:len(authversions.Migrations)]).
			To(Equal(authversions.Migrations))
	})
})
