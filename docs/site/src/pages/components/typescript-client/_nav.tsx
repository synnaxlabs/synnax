// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const typescriptClientNav: PageNavNode = {
  key: "typescript-client",
  name: "Typescript Client",
  children: [
    {
      key: "/components/typescript-client/get-started",
      href: "/components/typescript-client/get-started",
      name: "Get Started",
    },
    {
      key: "/components/typescript-client/channels",
      href: "/components/typescript-client/channels",
      name: "Channels",
    },
    {
      key: "/components/typescript-client/read-data",
      href: "/components/typescript-client/read-data",
      name: "Read Data",
    },
    {
      key: "/components/typescript-client/write-data",
      href: "/components/typescript-client/write-data",
      name: "Write Data",
    },
    {
      key: "/components/typescript-client/stream-data",
      href: "/components/typescript-client/stream-data",
      name: "Stream Data",
    },
    {
      key: "/components/typescript-client/series-and-frames",
      href: "/components/typescript-client/series-and-frames",
      name: "Series and Frames",
    },
    {
      key: "/components/typescript-client/examples",
      href: "/components/typescript-client/examples",
      name: "Examples",
    },
  ],
};
