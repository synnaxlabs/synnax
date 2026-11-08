// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const CHANNELS_NAV: Nav.PageNavNode = {
  key: "channels",
  name: "Channels",
  children: [
    {
      href: "/reference/channels",
      name: "Overview",
    },
    {
      href: "/reference/channels/create",
      name: "Create",
    },
    {
      href: "/reference/channels/find",
      name: "Find",
    },
    {
      href: "/reference/channels/rename-and-delete",
      name: "Rename and delete",
    },
    {
      href: "/reference/channels/calculated",
      name: "Calculated channels",
    },
    {
      href: "/reference/channels/reference",
      name: "Reference",
    },
  ],
};
