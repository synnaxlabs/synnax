// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const guidesNav: PageNavNode = {
  key: "guides",
  name: "Guides",
  children: [
    {
      name: "Exploratory Analysis in Python",
      href: "/guides/exploratory-analysis-in-python",
      key: "/guides/exploratory-analysis-in-python",
    },
    {
      name: "Automated Post Processing",
      href: "/guides/automated-post-processing",
      key: "/guides/automated-post-processing",
    },
    {
      name: "Writing Control Sequences",
      href: "/guides/writing-control-sequences",
      key: "/guides/writing-control-sequences",
    },
  ],
};
