// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const PYTHON_SEQUENCES_NAV: PageNavNode = {
  key: "python",
  name: "Python Sequences",
  children: [
    {
      key: "/reference/control/python/get-started",
      href: "/reference/control/python/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/control/python/examples",
      href: "/reference/control/python/examples",
      name: "Examples",
    },
  ],
};
