// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const CLIENT_NAV: PageNavNode = {
  key: "client",
  name: "Client libraries",
  children: [
    {
      href: "/reference/client/quick-start",
      name: "Quick start",
    },
    {
      href: "/reference/client/authentication",
      name: "Authentication",
    },
    {
      href: "/reference/client/channels",
      name: "Channels",
    },
    {
      href: "/reference/client/calculated-channels",
      name: "Calculated channels",
    },
    {
      href: "/reference/client/ranges",
      name: "Ranges",
    },
    {
      href: "/reference/client/queries",
      name: "Queries",
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
      href: "/reference/client/read-data",
      name: "Reading data",
    },
    {
      href: "/reference/client/write-data",
      name: "Writing data",
    },
    {
      key: "advanced",
      name: "Advanced topics",
      children: [
        {
          href: "/reference/client/advanced/auto-commit",
          name: "Auto-commit",
        },
        {
          href: "/reference/client/advanced/auto-index",
          name: "Auto-index",
        },
        {
          href: "/reference/client/advanced/write-authorities",
          name: "Write authorities",
        },
        {
          href: "/reference/client/advanced/iterators",
          name: "Iterators",
        },
        {
          href: "/reference/client/advanced/delete-data",
          name: "Delete data",
        },
        {
          href: "/reference/client/advanced/build-device-driver",
          name: "Build a device driver",
        },
      ],
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
