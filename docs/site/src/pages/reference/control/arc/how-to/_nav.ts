// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const HOW_TO_NAV: PageNavNode = {
  key: "how-to",
  name: "How-To Guides",
  children: [
    {
      key: "/reference/control/arc/how-to/data-processing",
      href: "/reference/control/arc/how-to/data-processing",
      name: "Data Processing",
    },
    {
      key: "/reference/control/arc/how-to/alarms",
      href: "/reference/control/arc/how-to/alarms",
      name: "Alarms",
    },
    {
      key: "/reference/control/arc/how-to/bang-bang-control",
      href: "/reference/control/arc/how-to/bang-bang-control",
      name: "Bang-Bang Control",
    },
    {
      key: "/reference/control/arc/how-to/test-sequences",
      href: "/reference/control/arc/how-to/test-sequences",
      name: "Test Sequences",
    },
  ],
};
