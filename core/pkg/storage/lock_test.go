// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage_test

import (
	"bufio"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

// locked matches storage.ErrLocked. MatchError uses the standard errors.Is, which does
// not see a mark.
func locked(err error) bool { return errors.Is(err, storage.ErrLocked) }

var _ = Describe("Data directory lock", func() {
	It(
		"Should return ErrLocked when another process holds the data",
		func(ctx SpecContext) {
			dir := GinkgoT().TempDir()
			holder := exec.Command(os.Args[0])
			holder.Env = append(os.Environ(), holdLockEnv+"="+dir)
			stdin := MustSucceed(holder.StdinPipe())
			stdout := MustSucceed(holder.StdoutPipe())
			Expect(holder.Start()).To(Succeed())
			DeferCleanup(func() {
				Expect(stdin.Close()).To(Succeed())
				Expect(holder.Wait()).To(Succeed())
			})
			Expect(bufio.NewReader(stdout).ReadString('\n')).To(Equal("locked\n"))
			cfg := storage.LayerConfig{Dirname: dir, Version: "0.60.0"}
			Expect(storage.OpenLayer(ctx, cfg)).Error().
				To(MatchError(locked, "storage.ErrLocked"))
		},
	)
})
