// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/text/unicode/norm"
)

func TestGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Group Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// nameRules gives the channel scope a rule that stores each name in NFC, as the rule
// that the service layer gives it does. The channel package specs the rest of that
// rule.
var nameRules = map[resource.Type]group.NameRule{
	resource.TypeChannel: func(name string) (string, error) {
		return norm.NFC.String(name), nil
	},
}
