// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const CLIENT_NAV: Nav.PageNavNode = {
  key: "client",
  name: "Client libraries",
  edition: "enterprise",
  children: [
    {
      href: "/reference/client",
      name: "Overview",
    },
    {
      href: "/reference/client/quick-start",
      name: "Quick start",
    },
    {
      href: "/reference/client/connect",
      name: "Connect",
    },
    {
      href: "/reference/client/series-and-frames",
      name: "Series and frames",
    },
    {
      href: "/reference/client/time-types",
      name: "Time types",
    },
    {
      href: "/reference/client/write-data",
      name: "Write data",
    },
    {
      href: "/reference/client/read-data",
      name: "Read data",
    },
    {
      href: "/reference/client/stream-data",
      name: "Stream data",
    },
    {
      href: "/reference/client/delete-data",
      name: "Delete data",
    },
    {
      href: "/reference/client/examples",
      name: "Examples",
    },
    {
      href: "/reference/client/troubleshooting",
      name: "Troubleshooting",
    },
  ],
};
