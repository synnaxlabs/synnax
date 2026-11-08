// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const STANDARD_LIBRARY_NAV: Nav.PageNavNode = {
  href: "/reference/control/arc/standard-library",
  name: "Standard library",
  children: [
    {
      href: "/reference/control/arc/standard-library/control",
      name: "`control`",
    },
    {
      href: "/reference/control/arc/standard-library/math",
      name: "`math`",
    },
    {
      href: "/reference/control/arc/standard-library/ranges",
      name: "`ranges`",
    },
    {
      href: "/reference/control/arc/standard-library/select",
      name: "`select`",
    },
    {
      href: "/reference/control/arc/standard-library/stable",
      name: "`stable`",
    },
    {
      href: "/reference/control/arc/standard-library/status",
      name: "`status`",
    },
    {
      href: "/reference/control/arc/standard-library/time",
      name: "`time`",
    },
  ],
};
