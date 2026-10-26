// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const STANDARD_LIBRARY_NAV: PageNavNode = {
  href: "/reference/control/arc/reference/standard-library",
  name: "Standard library",
  children: [
    {
      href: "/reference/control/arc/reference/standard-library/control",
      name: "`control`",
    },
    {
      href: "/reference/control/arc/reference/standard-library/math",
      name: "`math`",
    },
    {
      href: "/reference/control/arc/reference/standard-library/ranges",
      name: "`ranges`",
    },
    {
      href: "/reference/control/arc/reference/standard-library/select",
      name: "`select`",
    },
    {
      href: "/reference/control/arc/reference/standard-library/stable",
      name: "`stable`",
    },
    {
      href: "/reference/control/arc/reference/standard-library/status",
      name: "`status`",
    },
    {
      href: "/reference/control/arc/reference/standard-library/time",
      name: "`time`",
    },
  ],
};
