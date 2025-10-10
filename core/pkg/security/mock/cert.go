// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/testutil"
)

// SmallKeySize to run tests faster.
const SmallKeySize = 1024

func GenerateCerts(fs xfs.FS) {
	f := testutil.MustSucceed(cert.NewFactory(cert.FactoryConfig{
		LoaderConfig: cert.LoaderConfig{FS: fs},
		KeySize:      1024,
		Hosts:        []address.Address{"localhost:26260"},
	}))
	gomega.Expect(f.CreateCAPair()).To(gomega.Succeed())
	gomega.Expect(f.CreateNodePair()).To(gomega.Succeed())
}
