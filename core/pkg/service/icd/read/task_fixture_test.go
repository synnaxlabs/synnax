// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package read_test

import (
	"context"
	"encoding/json/v2"
	"io/fs"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// taskFixture is the config of a bus read task, with its ICDs and channels. The
// Driver's Receiver runs the same fixtures. An invalid one holds the error of the
// first problem and its path in the config.
type taskFixture struct {
	Description string          `json:"description"`
	Error       string          `json:"error"`
	Path        string          `json:"path"`
	ICDs        []icd.ICD       `json:"icds"`
	Channels    []taskChannel   `json:"channels"`
	Entries     []icd.ReadEntry `json:"entries"`
}

// taskChannel is a channel of a task fixture, in the form that the Driver parses.
type taskChannel struct {
	Key      channel.Key    `json:"key"`
	Name     string         `json:"name"`
	DataType telem.DataType `json:"data_type"`
	IsIndex  bool           `json:"is_index"`
	Index    channel.Key    `json:"index"`
}

var taskFixtures = os.DirFS("../testdata/task")

func taskFixtureEntries(kind string) []TableEntry {
	GinkgoHelper()
	paths := MustSucceed(fs.Glob(taskFixtures, kind+"/*.json"))
	return lo.Map(paths, func(path string, _ int) TableEntry {
		name := strings.TrimSuffix(strings.TrimPrefix(path, kind+"/"), ".json")
		return Entry(name, path)
	})
}

// checkTask checks the entries of the fixture at path, with its ICDs and channels.
func checkTask(ctx context.Context, path string) (taskFixture, error) {
	GinkgoHelper()
	var f taskFixture
	Expect(json.Unmarshal(MustSucceed(fs.ReadFile(taskFixtures, path)), &f)).
		To(Succeed())
	Expect(f.Description).ToNot(BeEmpty())
	channels := lo.SliceToMap(
		f.Channels,
		func(ch taskChannel) (channel.Key, icdv0.Channel) {
			return ch.Key, icdv0.Channel{
				Name:     ch.Name,
				DataType: ch.DataType,
				Index:    ch.IsIndex,
				IndexKey: ch.Index,
			}
		},
	)
	resolver := read.NewResolver(icdSvc, func(
		context.Context,
		gorp.Snapshot,
		[]channel.Key,
	) (map[channel.Key]icdv0.Channel, error) {
		return channels, nil
	})
	icds := lo.SliceToMap(f.ICDs, func(ic icd.ICD) (icd.Key, icd.ICD) {
		return ic.Key, ic
	})
	return f, resolver.Check(ctx, db, read.Entries(f.Entries), icds)
}

var _ = Describe("Task fixtures", func() {
	DescribeTable("Should accept a valid bus read task", func(
		ctx SpecContext,
		path string,
	) {
		Expect(MustSucceed(checkTask(ctx, path)).Error).To(BeEmpty())
	}, taskFixtureEntries("valid"))

	DescribeTable("Should refuse an invalid bus read task with its error at its path",
		func(ctx SpecContext, path string) {
			f, err := checkTask(ctx, path)
			Expect(err).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(f.Path+": "+f.Error),
			))
		},
		taskFixtureEntries("invalid"),
	)
})
