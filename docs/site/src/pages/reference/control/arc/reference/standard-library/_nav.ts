// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const STANDARD_LIBRARY_NAV: PageNavNode = {
  key: "standard-library",
  href: "/reference/control/arc/reference/standard-library",
  name: "Standard Library",
  children: [
    {
      key: "/reference/control/arc/reference/standard-library/control",
      href: "/reference/control/arc/reference/standard-library/control",
      name: "Control",
    },
    {
      key: "/reference/control/arc/reference/standard-library/math",
      href: "/reference/control/arc/reference/standard-library/math",
      name: "Math",
    },
    {
      key: "/reference/control/arc/reference/standard-library/ranges",
      href: "/reference/control/arc/reference/standard-library/ranges",
      name: "Ranges",
    },
    {
      key: "/reference/control/arc/reference/standard-library/status",
      href: "/reference/control/arc/reference/standard-library/status",
      name: "Status",
    },
    {
      key: "/reference/control/arc/reference/standard-library/time",
      href: "/reference/control/arc/reference/standard-library/time",
      name: "Time",
    },
  ],
};
