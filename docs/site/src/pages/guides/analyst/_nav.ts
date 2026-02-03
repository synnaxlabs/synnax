// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const ANALYST_NAV: PageNavNode = {
  key: "analyst",
  name: "Analysts",
  icon: "Analyze",
  children: [
    {
      name: "Exploratory Analysis in Python",
      href: "/guides/analyst/exploratory-analysis-in-python",
      key: "/guides/analyst/exploratory-analysis-in-python",
    },
  ],
};
