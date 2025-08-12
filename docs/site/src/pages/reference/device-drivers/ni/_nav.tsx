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
      key: "/reference/device-drivers/ni/get-started",
      href: "/reference/device-drivers/ni/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/device-drivers/ni/configure-device",
      href: "/reference/device-drivers/ni/configure-device",
      name: "Configure a Device",
    },
    {
      key: "/reference/device-drivers/ni/analog-read-task",
      href: "/reference/device-drivers/ni/analog-read-task",
      name: "Analog Read Task",
    },
    {
      key: "/reference/device-drivers/ni/analog-write-task",
      href: "/reference/device-drivers/ni/analog-write-task",
      name: "Analog Write Task",
    },
    {
      key: "/reference/device-drivers/ni/digital-read-task",
      href: "/reference/device-drivers/ni/digital-read-task",
      name: "Digital Read Task",
    },
    {
      key: "/reference/device-drivers/ni/digital-write-task",
      href: "/reference/device-drivers/ni/digital-write-task",
      name: "Digital Write Task",
    },
  ],
};
