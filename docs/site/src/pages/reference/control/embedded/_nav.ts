// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const EMBEDDED_SEQUENCES_NAV: PageNavNode = {
  key: "embedded",
  name: "Embedded Sequences",
  children: [
    {
      key: "/reference/control/embedded/get-started",
      href: "/reference/control/embedded/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/control/embedded/recipes",
      href: "/reference/control/embedded/recipes",
      name: "Recipes",
    },
    {
      key: "/reference/control/embedded/reference",
      href: "/reference/control/embedded/reference",
      name: "Reference",
    },
  ],
};
