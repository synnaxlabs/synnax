// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var (
	db     *gorp.DB
	mockFS fs.FS
	keys   *apikey.Service
)

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// Process #1 compiles the mock Driver once and hands its directory to the others.
// Linking it is the suite's largest memory and disk cost, so it runs once rather than
// once per parallel process.
var _ = SynchronizedBeforeSuite(func() []byte {
	ShouldNotLeakGoroutines()
	dir := MustSucceed(os.MkdirTemp("", "mockdriver"))
	// Ginkgo holds this until every other process finishes, so no process reads the
	// binary after it is removed.
	DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	driverName := "driver"
	if runtime.GOOS == "windows" {
		driverName = "driver.exe"
	}
	cmd := exec.Command(
		"go", "build", "-o", filepath.Join(dir, driverName),
		"./testdata/mockdriver",
	)
	cmd.Dir = MustSucceed(os.Getwd())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Expect(cmd.Run()).To(Succeed())
	return []byte(dir)
}, func(ctx SpecContext, dir []byte) {
	mockFS = os.DirFS(string(dir))
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[rack.Key, rack.Rack]{
		DB:   db,
		Name: string(resource.TypeRack),
	}))
	r := rack.Rack{Key: rackKey, Name: "driver"}
	Expect(racks.NewCreate().Entry(&r).Exec(ctx, db)).To(Succeed())
	keys = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
})
