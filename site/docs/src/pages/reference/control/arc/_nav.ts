// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";
import { REFERENCE_NAV } from "@/pages/reference/control/arc/reference/_nav";

export const ARC_NAV: PageNavNode = {
  key: "arc",
  name: "Arc",
  children: [
    {
      key: "/reference/control/arc",
      href: "/reference/control/arc",
      name: "Overview",
    },
    {
      key: "/reference/control/arc/get-started",
      href: "/reference/control/arc/get-started",
      name: "Get started",
    },
    {
      key: "/reference/control/arc/execution-model",
      href: "/reference/control/arc/execution-model",
      name: "Execution model",
    },
    {
      key: "/reference/control/arc/timing",
      href: "/reference/control/arc/timing",
      name: "Timing",
    },
    {
      key: "/reference/control/arc/automations",
      href: "/reference/control/arc/automations",
      name: "Automations",
    },
    {
      key: "/reference/control/arc/graph-editor",
      href: "/reference/control/arc/graph-editor",
      name: "Graph editor",
    },
    {
      key: "/reference/control/arc/start-from-schematic",
      href: "/reference/control/arc/start-from-schematic",
      name: "Start from a schematic",
    },
    {
      key: "/reference/control/arc/test-sequences",
      href: "/reference/control/arc/test-sequences",
      name: "Test sequences",
    },
    {
      key: "/reference/control/arc/bang-bang-control",
      href: "/reference/control/arc/bang-bang-control",
      name: "Bang-bang control",
    },
    {
      key: "/reference/control/arc/compute-values",
      href: "/reference/control/arc/compute-values",
      name: "Compute values",
    },
    {
      key: "/reference/control/arc/effective-arc",
      href: "/reference/control/arc/effective-arc",
      name: "Effective Arc",
    },
    REFERENCE_NAV,
  ],
};
