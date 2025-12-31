// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const CLIENT_NAV: PageNavNode = {
  key: "client",
  name: "Client",
  children: [
    {
      key: "/reference/client/quick-start",
      href: "/reference/client/quick-start",
      name: "Quick Start",
    },
    {
      key: "/reference/client/authentication",
      href: "/reference/client/authentication",
      name: "Authentication",
    },
    {
      key: "/reference/client/channels",
      href: "/reference/client/channels",
      name: "Channels",
    },
    {
      key: "/reference/client/ranges",
      href: "/reference/client/ranges",
      name: "Ranges",
    },
    {
      key: "/reference/client/series-and-frames",
      href: "/reference/client/series-and-frames",
      name: "Series & Frames",
    },
    {
      key: "/reference/client/time-types",
      href: "/reference/client/time-types",
      name: "Time Types",
    },
    {
      key: "/reference/client/read-data",
      href: "/reference/client/read-data",
      name: "Reading Data",
    },
    {
      key: "/reference/client/write-data",
      href: "/reference/client/write-data",
      name: "Writing Data",
    },
    {
      key: "advanced",
      name: "Advanced Topics",
      children: [
        {
          key: "/reference/client/advanced/auto-commit",
          href: "/reference/client/advanced/auto-commit",
          name: "Auto-Commit",
        },
        {
          key: "/reference/client/advanced/write-authorities",
          href: "/reference/client/advanced/write-authorities",
          name: "Write Authorities",
        },
        {
          key: "/reference/client/advanced/iterators",
          href: "/reference/client/advanced/iterators",
          name: "Iterators",
        },
        {
          key: "/reference/client/advanced/delete-data",
          href: "/reference/client/advanced/delete-data",
          name: "Delete Data",
        },
        {
          key: "/reference/client/advanced/build-device-driver",
          href: "/reference/client/advanced/build-device-driver",
          name: "Build a Device Driver",
        },
      ],
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
