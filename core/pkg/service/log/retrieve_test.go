// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
)

var _ = Describe("Retrieve", func() {
	It("Should retrieve a Log", func(ctx SpecContext) {
		l := log.Log{
			Name: "test",
			Channels: []log.ChannelEntry{
				{Channel: channel.Key(1), Color: color.MustFromHex("#ff0000"), Notation: notation.NotationStandard},
			},
			ShowChannelNames: true,
		}
		Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
		var res log.Log
		Expect(svc.NewRetrieve().Where(log.MatchKeys(l.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
		Expect(res).To(Equal(l))
	})
})
