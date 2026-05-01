// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flagdef_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/flagdef"
	. "github.com/synnaxlabs/x/testutil"
)

func TestFlagdef(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flagdef Suite")
}

var _ = Describe("Flagdef", func() {
	Describe("Parse", func() {
		It("Should parse all supported flag types", func() {
			defs := MustSucceed(flagdef.Parse([]byte(`[
				{"name": "s", "type": "string", "default": "v"},
				{"name": "b", "type": "bool", "default": true},
				{"name": "i", "type": "int", "default": 7},
				{"name": "d", "type": "duration", "default": "5s"},
				{"name": "ss", "type": "stringSlice", "default": ["a", "b"]}
			]`)))
			Expect(defs).To(HaveLen(5))
		})

		It("Should return an error on malformed JSON", func() {
			Expect(flagdef.Parse([]byte("{not json"))).Error().
				To(MatchError(ContainSubstring("parse flag definitions")))
		})
	})

	Describe("Register", func() {
		It("Should register flags of every supported type with correct defaults", func() {
			defs := MustSucceed(flagdef.Parse([]byte(`[
				{"name": "host", "short": "h", "type": "string", "default": "localhost"},
				{"name": "verbose", "short": "v", "type": "bool", "default": false},
				{"name": "count", "type": "int", "default": 4},
				{"name": "timeout", "type": "duration", "default": "2.5s"},
				{"name": "peers", "short": "p", "type": "stringSlice", "default": ["a"]}
			]`)))
			cmd := &cobra.Command{Use: "test"}
			Expect(flagdef.Register(cmd, defs)).To(Succeed())

			Expect(MustSucceed(cmd.Flags().GetString("host"))).To(Equal("localhost"))
			Expect(cmd.Flags().Lookup("host").Shorthand).To(Equal("h"))
			Expect(MustSucceed(cmd.Flags().GetBool("verbose"))).To(BeFalse())
			Expect(MustSucceed(cmd.Flags().GetInt("count"))).To(Equal(4))
			Expect(MustSucceed(cmd.Flags().GetDuration("timeout"))).
				To(Equal(2500 * time.Millisecond))
			Expect(MustSucceed(cmd.Flags().GetStringSlice("peers"))).To(Equal([]string{"a"}))
		})

		It("Should register persistent flags on the persistent flag set", func() {
			defs := MustSucceed(flagdef.Parse([]byte(`[
				{"name": "config", "type": "string", "default": "c.yaml", "persistent": true},
				{"name": "version", "type": "bool", "default": false}
			]`)))
			cmd := &cobra.Command{Use: "test"}
			Expect(flagdef.Register(cmd, defs)).To(Succeed())
			Expect(cmd.PersistentFlags().Lookup("config")).ToNot(BeNil())
			Expect(cmd.PersistentFlags().Lookup("version")).To(BeNil())
		})

		It("Should reject unsupported flag types", func() {
			defs := []flagdef.Definition{{Name: "x", Type: "uint64", Default: 0}}
			cmd := &cobra.Command{Use: "test"}
			Expect(flagdef.Register(cmd, defs)).To(MatchError(ContainSubstring(
				`unsupported flag type "uint64"`,
			)))
		})

		It("Should reject an unparseable duration default", func() {
			defs := []flagdef.Definition{{Name: "x", Type: "duration", Default: "nope"}}
			cmd := &cobra.Command{Use: "test"}
			Expect(flagdef.Register(cmd, defs)).To(MatchError(ContainSubstring(
				`parse duration "nope"`,
			)))
		})
	})
})
