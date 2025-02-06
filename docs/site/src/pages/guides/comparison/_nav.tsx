// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const comparisonNav: PageNavNode = {
  key: "comparison",
  name: "Comparisons",
  children: [
    {
      name: "Database Performance",
      href: "/guides/comparison/performance/one-billion-rows",
      key: "/guides/comparison/performance/one-billion-rows",
    },
  ],
};
