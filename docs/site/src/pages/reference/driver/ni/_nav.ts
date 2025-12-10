// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const NI_NAV: PageNavNode = {
  key: "ni",
  name: "National Instruments",
  children: [
    {
      key: "/reference/driver/ni/get-started",
      href: "/reference/driver/ni/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/ni/configure-device",
      href: "/reference/driver/ni/configure-device",
      name: "Configure a Device",
    },
    {
      key: "/reference/driver/ni/analog-read-task",
      href: "/reference/driver/ni/analog-read-task",
      name: "Analog Read Task",
    },
    {
      key: "/reference/driver/ni/analog-write-task",
      href: "/reference/driver/ni/analog-write-task",
      name: "Analog Write Task",
    },
    {
      key: "/reference/driver/ni/counter-read-task",
      href: "/reference/driver/ni/counter-read-task",
      name: "Counter Read Task",
    },
    {
      key: "/reference/driver/ni/digital-read-task",
      href: "/reference/driver/ni/digital-read-task",
      name: "Digital Read Task",
    },
    {
      key: "/reference/driver/ni/digital-write-task",
      href: "/reference/driver/ni/digital-write-task",
      name: "Digital Write Task",
    },
  ],
};
