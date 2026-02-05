// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";
import { CONCEPTS_NAV } from "@/pages/reference/control/arc/concepts/_nav";
import { HOW_TO_NAV } from "@/pages/reference/control/arc/how-to/_nav";
import { REFERENCE_NAV } from "@/pages/reference/control/arc/reference/_nav";

export const ARC_NAV: PageNavNode = {
  key: "arc",
  name: "Arc",
  children: [
    {
      key: "/reference/control/arc/introduction",
      href: "/reference/control/arc/introduction",
      name: "Introduction",
    },
    {
      key: "/reference/control/arc/get-started",
      href: "/reference/control/arc/get-started",
      name: "Get Started",
    },
    CONCEPTS_NAV,
    HOW_TO_NAV,
    {
      key: "/reference/control/arc/effective-arc",
      href: "/reference/control/arc/effective-arc",
      name: "Effective Arc",
    },
    REFERENCE_NAV,
  ],
};
