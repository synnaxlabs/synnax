// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

const repositoryRoot = "../.."

// repositoryChannels holds the channels that the .arc files in the repository use. A
// new file that uses a new channel adds it here.
var repositoryChannels = map[string]types.Type{
	"sample_tank_pt":         types.F64(),
	"sample_press_vlv_state": types.U8(),
	"sample_vent_vlv_state":  types.U8(),
	"sample_press_vlv_cmd":   types.U8(),
	"sample_vent_vlv_cmd":    types.U8(),
	"sample_start_cmd":       types.U8(),
	"sample_abort_cmd":       types.U8(),
	"start_sim_cmd":          types.U8(),
	"start_seq_cmd":          types.U8(),
	"ox_pt_1":                types.F32(),
	"ox_pt_2":                types.F32(),
	"ox_tc_1":                types.F32(),
	"press_pt_1":             types.F32(),
	"ox_mpv_cmd":             types.U8(),
	"ox_vent_cmd":            types.U8(),
	"ox_vent_state":          types.U8(),
	"ox_tpc_cmd":             types.U8(),
	"ox_press_cmd":           types.U8(),
	"ox_press_state":         types.U8(),
	"press_iso_cmd":          types.U8(),
	"gas_booster_iso_cmd":    types.U8(),
	"fuel_mpv_cmd":           types.U8(),
}

func findRepositoryPrograms() ([]string, error) {
	var paths []string
	err := filepath.WalkDir(
		repositoryRoot,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() && path != repositoryRoot &&
				(strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			if !d.IsDir() && filepath.Ext(name) == ".arc" {
				paths = append(paths, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func newRepositoryRoot() *symbol.Symbol {
	channels := make([]*symbol.Symbol, 0, len(repositoryChannels))
	id := 1
	for name, t := range repositoryChannels {
		channels = append(channels, &symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(t),
			ID:   id,
		})
		id++
	}
	return arc.NewRoot(nil, channels...)
}

var _ = Describe("Repository programs", func() {
	paths := MustSucceed(findRepositoryPrograms())
	It("should find the .arc files in the repository", func() {
		Expect(paths).ToNot(BeEmpty())
	})
	DescribeTable(
		"should compile",
		func(ctx SpecContext, path string) {
			raw := string(MustSucceed(os.ReadFile(path)))
			MustSucceed(arc.CompileText(ctx, arc.Text{Raw: raw}, newRepositoryRoot()))
		},
		lo.Map(paths, func(path string, _ int) TableEntry { return Entry(path, path) }),
	)
})
