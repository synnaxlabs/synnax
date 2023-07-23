// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { PageNavLeaf } from "@/components/PageNav";

export const analyzeNav: PageNavLeaf = {
  key: "analyze",
  name: "Analyze",
  icon: <Icon.Analyze />,
  children: [
    {
      key: "/analyze/get-started",
      url: "/analyze/get-started",
      name: "Get Started",
    },
    {
      key: "/analyze/retrieve-channels",
      url: "/analyze/retrieve-channels",
      name: "Retrieve Channels",
    },
    {
      key: "/analyze/read-telemetry",
      url: "/analyze/read-telemetry",
      name: "Read Telemetry",
    },
  ],
};
