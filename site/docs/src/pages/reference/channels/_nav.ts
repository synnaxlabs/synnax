// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const CHANNELS_NAV: PageNavNode = {
  key: "channels",
  name: "Channels",
  children: [
    {
      key: "/reference/channels",
      href: "/reference/channels",
      name: "Overview",
    },
    {
      key: "/reference/channels/create",
      href: "/reference/channels/create",
      name: "Create channels",
    },
    {
      key: "/reference/channels/find",
      href: "/reference/channels/find",
      name: "Find channels",
    },
    {
      key: "/reference/channels/rename-and-delete",
      href: "/reference/channels/rename-and-delete",
      name: "Rename and delete channels",
    },
    {
      key: "/reference/channels/calculated",
      href: "/reference/channels/calculated",
      name: "Calculated channels",
    },
  ],
};
