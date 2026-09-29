// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";
import { ARC_NAV } from "@/pages/reference/control/arc/_nav";
import { PYTHON_SEQUENCES_NAV } from "@/pages/reference/control/python/_nav";

export const CONTROL_NAV: PageNavNode = {
  key: "control",
  name: "Control",
  children: [
    {
      key: "/reference/control/get-started",
      href: "/reference/control/get-started",
      name: "Get started",
    },
    {
      key: "/reference/control/manual",
      href: "/reference/control/manual",
      name: "Manual control",
    },
    {
      key: "/reference/control/authority",
      href: "/reference/control/authority",
      name: "Control authority",
    },
    ARC_NAV,
    PYTHON_SEQUENCES_NAV,
  ],
};
