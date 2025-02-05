// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const controlNav: PageNavNode = {
  key: "control",
  name: "Control",
  children: [
    {
      key: "/reference/control/get-started",
      href: "/reference/control/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/control/sequence-basics",
      href: "/reference/control/sequence-basics",
      name: "Sequence Basics",
    },
    {
      key: "/reference/control/examples",
      href: "/reference/control/examples",
      name: "Examples",
    },
  ],
};
