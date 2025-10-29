// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const OPERATIONS_NAV: PageNavNode = {
  key: "operations",
  name: "Test and Operations",
  children: [
    {
      name: "Writing Automated Control Sequences",
      href: "/guides/operations/automated-control",
      key: "/guides/operations/automated-control",
    },
  ],
};
