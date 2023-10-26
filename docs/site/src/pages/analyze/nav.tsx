// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const analyzeNav: PageNavNode = {
  key: "analyze",
  name: "Analyze",
  children: [
    {
      key: "/analyze/get-started",
      href: "/analyze/get-started",
      name: "Get Started",
    },
    {
      key: "/analyze/retrieve-channels",
      href: "/analyze/retrieve-channels",
      name: "Retrieve Channels",
    },
    {
      key: "/analyze/read-telemetry",
      href: "/analyze/read-telemetry",
      name: "Read Telemetry",
    },
    {
      key: "/analyze/named-ranges",
      href: "/analyze/named-ranges",
      name: "Named Ranges",
    },
  ],
};
