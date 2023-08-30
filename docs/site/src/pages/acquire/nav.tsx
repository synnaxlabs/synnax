// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { PageNavNode } from "@/components/PageNav";

export const acquireNav: PageNavNode = {
  key: "acquire",
  name: "Acquire",
  icon: <Icon.Acquire />,
  children: [
    {
      key: "/acquire/get-started",
      href: "/acquire/get-started",
      name: "Get Started",
    },
    {
      key: "/acquire/create-channels",
      href: "/acquire/create-channels",
      name: "Create Channels",
    },
    {
      key: "/acquire/write-telemetry",
      href: "/acquire/write-telemetry",
      name: "Write Telemetry",
    },
    {
      key: "/acquire/ingest-files",
      href: "/acquire/ingest-files",
      name: "Ingest Files",
    },
  ],
};
