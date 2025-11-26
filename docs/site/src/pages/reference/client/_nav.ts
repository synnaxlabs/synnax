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
      key: "fundamentals",
      name: "Fundamentals",
      children: [
        {
          key: "/reference/client/fundamentals/channels",
          href: "/reference/client/fundamentals/channels",
          name: "Channels",
        },
        {
          key: "/reference/client/fundamentals/read-data",
          href: "/reference/client/fundamentals/read-data",
          name: "Reading Data",
        },
        {
          key: "/reference/client/fundamentals/write-data",
          href: "/reference/client/fundamentals/write-data",
          name: "Writing Data",
        },
      ],
    },
    {
      key: "working-with-data",
      name: "Working with Data",
      children: [
        {
          key: "/reference/client/working-with-data/series-and-frames",
          href: "/reference/client/working-with-data/series-and-frames",
          name: "Series & Frames",
        },
        {
          key: "/reference/client/working-with-data/ranges",
          href: "/reference/client/working-with-data/ranges",
          name: "Ranges",
        },
        {
          key: "/reference/client/working-with-data/streaming-data",
          href: "/reference/client/working-with-data/streaming-data",
          name: "Streaming Data",
        },
        {
          key: "/reference/client/working-with-data/iterators",
          href: "/reference/client/working-with-data/iterators",
          name: "Iterators",
        },
      ],
    },
    {
      key: "advanced",
      name: "Advanced Topics",
      children: [
        {
          key: "/reference/client/advanced/writers",
          href: "/reference/client/advanced/writers",
          name: "Writers",
        },
        {
          key: "/reference/client/advanced/delete-data",
          href: "/reference/client/advanced/delete-data",
          name: "Delete Data",
        },
        {
          key: "/reference/client/advanced/timestamps",
          href: "/reference/client/advanced/timestamps",
          name: "Timestamps",
        },
      ],
    },
    {
      key: "resources",
      name: "Resources",
      children: [
        {
          key: "/reference/client/resources/examples",
          href: "/reference/client/resources/examples",
          name: "Examples",
        },
        {
          key: "/reference/client/resources/build-device-driver",
          href: "/reference/client/resources/build-device-driver",
          name: "Build a Device Driver",
        },
      ],
    },
  ],
};
