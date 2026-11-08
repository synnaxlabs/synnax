// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const NI_NAV: Nav.PageNavNode = {
  key: "ni",
  name: "National Instruments",
  children: [
    {
      href: "/reference/hardware/ni/configure-device",
      name: "Configure a device",
    },
    {
      href: "/reference/hardware/ni/analog-read-task",
      name: "Analog read task",
    },
    {
      href: "/reference/hardware/ni/analog-write-task",
      name: "Analog write task",
    },
    {
      href: "/reference/hardware/ni/counter-read-task",
      name: "Counter read task",
    },
    {
      href: "/reference/hardware/ni/digital-read-task",
      name: "Digital read task",
    },
    {
      href: "/reference/hardware/ni/digital-write-task",
      name: "Digital write task",
    },
  ],
};
