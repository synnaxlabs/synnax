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
      key: "/reference/client/get-started",
      href: "/reference/client/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/client/channels",
      href: "/reference/client/channels",
      name: "Channels",
    },
    {
      key: "/reference/client/read-data",
      href: "/reference/client/read-data",
      name: "Read Data",
    },
    {
      key: "/reference/client/write-data",
      href: "/reference/client/write-data",
      name: "Write Data",
    },
  ],
};
