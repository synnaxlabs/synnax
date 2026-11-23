// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package session_test

import (
	"encoding/json/v2"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/imex/session"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func readAll(resource.ID) bool { return true }

var _ = Describe("Session", func() {
	var (
		l    *service.Layer
		db   *gorp.DB
		proj project.Project
		ch   channel.Channel
		lg   log.Log
		// readNoChannel lets the subject retrieve every record but ch.
		readNoChannel func(resource.ID) bool
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		l = MustOpen(svctestutil.OpenLayer(ctx, node))
		ch = channel.Channel{
			Name:     channeltestutil.UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		proj = project.Project{Name: "Stand"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := l.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			if err := l.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			lg = defaults.Fill(log.Log{Name: "Pressures", Project: &proj.Key})
			lg.Channels = []log.ChannelEntry{defaults.Fill(log.ChannelEntry{
				Channel:  ch.Key(),
				Notation: notation.NotationStandard,
			})}
			return l.Log.NewWriter(tx).Create(ctx, &lg)
		})).To(Succeed())
		hidden := channel.ResourceID(ch.Key())
		readNoChannel = func(id resource.ID) bool { return id != hidden }
	})

	// exportLog returns the log's file as an Export with readable writes it.
	exportLog := func(ctx SpecContext, readable func(resource.ID) bool) []byte {
		GinkgoHelper()
		exp := session.NewExport(ctx, l.ImEx, l.Channel, db, readable)
		env := MustSucceed(exp.Export(log.ResourceID(lg.Key)))
		return MustSucceed(json.Marshal(env))
	}

	Describe("Export", func() {
		It("Should write a channel link as the channel's path", func(ctx SpecContext) {
			Expect(string(exportLog(ctx, readAll))).
				To(ContainSubstring(fmt.Sprintf(`"channel":%q`, ch.Name)))
		})

		It("Should write a channel the exporter cannot retrieve as its raw key", func(
			ctx SpecContext,
		) {
			file := string(exportLog(ctx, readNoChannel))
			Expect(file).To(ContainSubstring(fmt.Sprintf(`"channel":%d`, ch.Key())))
			Expect(file).ToNot(ContainSubstring(ch.Name))
		})
	})

	Describe("Import", func() {
		decode := func(
			ctx SpecContext,
			readable func(resource.ID) bool,
		) (*session.Import, imex.Write) {
			GinkgoHelper()
			var env imex.Envelope
			Expect(json.Unmarshal(exportLog(ctx, readAll), &env)).To(Succeed())
			imp := session.NewImport(ctx, l.ImEx, l.Channel, db, readable)
			return imp, MustSucceed(imp.Decode(env, "Pressures.json"))
		}

		It("Should resolve a channel path to the channel's key", func(ctx SpecContext) {
			imp, write := decode(ctx, readAll)
			Expect(imp.Err()).To(Succeed())
			var id resource.ID
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				id, err = write(ctx, tx, imex.ImportOptions{
					Parent: project.ResourceID(proj.Key),
				})
				return err
			})).To(Succeed())
			var imported log.Log
			Expect(l.Log.NewRetrieve().
				Where(log.MatchKeys(uuid.MustParse(id.Key))).
				Entry(&imported).
				Exec(ctx, db)).To(Succeed())
			Expect(imported.Channels).To(HaveLen(1))
			Expect(imported.Channels[0].Channel).To(Equal(ch.Key()))
		})

		It("Should report a path to a channel the importer cannot retrieve", func(
			ctx SpecContext,
		) {
			imp, _ := decode(ctx, readNoChannel)
			Expect(imp.Err()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("no channel is at %q", ch.Name)),
			))
		})
	})
})
