// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";
import { ARC_NAV } from "@/pages/reference/control/arc/_nav";
import { PYTHON_SEQUENCES_NAV } from "@/pages/reference/control/python/_nav";

export const CONTROL_NAV: Nav.PageNavNode = {
  key: "control",
  name: "Control",
  children: [
    {
      href: "/reference/control",
      name: "Overview",
    },
    {
      href: "/reference/control/authority",
      name: "Authority",
    },
    {
      href: "/reference/control/manual",
      name: "Manual control",
    },
    ARC_NAV,
    PYTHON_SEQUENCES_NAV,
    {
      href: "/reference/control/troubleshooting",
      name: "Troubleshooting",
    },
  ],
};
