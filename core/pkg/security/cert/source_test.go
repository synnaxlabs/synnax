// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	. "github.com/synnaxlabs/x/testutil"
)

type stubSource struct{ cert.Source }

type stubFactory struct{ typ string }

func (f stubFactory) Type() string { return f.typ }

func (f stubFactory) NewSource(cert.SourceConfig) (cert.Source, error) {
	return stubSource{}, nil
}

var _ = Describe("Resolve", func() {
	factories := []cert.SourceFactory{stubFactory{"file"}, stubFactory{"auto"}}

	It("Should build the source whose type matches the config", func() {
		src := MustSucceed(cert.Resolve(factories, cert.SourceConfig{Type: "auto"}))
		Expect(src).To(Equal(stubSource{}))
	})

	It("Should reject an unknown source type", func() {
		Expect(cert.Resolve(factories, cert.SourceConfig{Type: "nope"})).
			Error().To(MatchError(ContainSubstring("unknown certificate source")))
	})
})
