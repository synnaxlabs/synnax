// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";
import { REFERENCE_NAV } from "@/pages/reference/control/arc/reference/_nav";
import { STANDARD_LIBRARY_NAV } from "@/pages/reference/control/arc/standard-library/_nav";

export const ARC_NAV: Nav.PageNavNode = {
  key: "arc",
  name: "Arc",
  children: [
    {
      href: "/reference/control/arc",
      name: "Overview",
    },
    {
      href: "/reference/control/arc/get-started",
      name: "Get started",
      edition: "desktop",
    },
    {
      href: "/reference/control/arc/execution-model",
      name: "Execution model",
    },
    {
      href: "/reference/control/arc/timing",
      name: "Timing",
    },
    {
      href: "/reference/control/arc/automations",
      name: "Automations",
    },
    {
      href: "/reference/control/arc/graph-editor",
      name: "Graph editor",
    },
    {
      href: "/reference/control/arc/start-from-schematic",
      name: "Start from a schematic",
    },
    {
      href: "/reference/control/arc/test-sequences",
      name: "Test sequences",
    },
    {
      href: "/reference/control/arc/bang-bang-control",
      name: "Bang-bang control",
    },
    {
      href: "/reference/control/arc/compute-values",
      name: "Compute values",
    },
    {
      href: "/reference/control/arc/effective-arc",
      name: "Effective Arc",
    },
    REFERENCE_NAV,
    STANDARD_LIBRARY_NAV,
  ],
};
