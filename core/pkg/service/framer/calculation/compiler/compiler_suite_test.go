// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:console/src/settings/ClusterTab.tsx
import { Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export const ClusterTab = (): ReactElement => (
  <Text.Text level="h4">Cluster Configuration - Coming Soon</Text.Text>
);
========
package compiler_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCompiler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compiler Suite")
}
>>>>>>>> 37de8639fa56d9208fe350e4b1b00e0e1a58a0bc:core/pkg/service/framer/calculation/compiler/compiler_suite_test.go
