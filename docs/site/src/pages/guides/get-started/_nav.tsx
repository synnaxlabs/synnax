// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const getStartedNav: PageNavNode = {
  key: "get-started",
  name: "Get Started",
  children: [
    {
      key: "/guides/get-started/installation",
      href: "/guides/get-started/installation",
      name: "Installation",
    },
  ],
};
