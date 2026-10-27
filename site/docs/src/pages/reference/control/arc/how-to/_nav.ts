// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const HOW_TO_NAV: PageNavNode = {
  key: "how-to",
  name: "How-to guides",
  children: [
    {
      href: "/reference/control/arc/how-to/data-processing",
      name: "Data processing",
    },
    {
      href: "/reference/control/arc/how-to/alarms",
      name: "Alarms",
    },
    {
      href: "/reference/control/arc/how-to/bang-bang-control",
      name: "Bang-bang control",
    },
    {
      href: "/reference/control/arc/how-to/test-sequences",
      name: "Test sequences",
    },
  ],
};
