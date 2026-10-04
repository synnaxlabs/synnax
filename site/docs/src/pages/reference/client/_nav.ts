// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const CLIENT_NAV: PageNavNode = {
  key: "client",
  name: "Client libraries",
  icon: "Terminal",
  edition: "enterprise",
  children: [
    {
      key: "/reference/client",
      href: "/reference/client",
      name: "Overview",
    },
    {
      key: "/reference/client/quick-start",
      href: "/reference/client/quick-start",
      name: "Quick start",
    },
    {
      key: "/reference/client/authentication",
      href: "/reference/client/authentication",
      name: "Authentication",
    },
    {
      key: "/reference/client/series-and-frames",
      href: "/reference/client/series-and-frames",
      name: "Series and frames",
    },
    {
      key: "/reference/client/time-types",
      href: "/reference/client/time-types",
      name: "Time types",
    },
    {
      key: "/reference/client/stream-data",
      href: "/reference/client/stream-data",
      name: "Streaming data",
    },
    {
      key: "/reference/client/read-data",
      href: "/reference/client/read-data",
      name: "Reading data",
    },
    {
      key: "/reference/client/write-data",
      href: "/reference/client/write-data",
      name: "Writing data",
    },
    {
      key: "advanced",
      name: "Advanced topics",
      children: [
        {
          key: "/reference/client/advanced/auto-commit",
          href: "/reference/client/advanced/auto-commit",
          name: "Auto-commit",
        },
        {
          key: "/reference/client/advanced/auto-index",
          href: "/reference/client/advanced/auto-index",
          name: "Auto-index",
        },
        {
          key: "/reference/client/advanced/iterators",
          href: "/reference/client/advanced/iterators",
          name: "Iterators",
        },
      ],
    },
    {
      key: "/reference/client/delete-data",
      href: "/reference/client/delete-data",
      name: "Delete data",
    },
    {
      key: "/reference/client/examples",
      href: "/reference/client/examples",
      name: "Examples",
    },
    {
      key: "/reference/client/troubleshooting",
      href: "/reference/client/troubleshooting",
      name: "Troubleshooting",
    },
  ],
};
