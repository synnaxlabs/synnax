// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";
import { labjackNav } from "@/pages/reference/device-drivers/labjack/_nav";
import { niNav } from "@/pages/reference/device-drivers/ni/_nav";
import { opcuaNav } from "@/pages/reference/device-drivers/opc-ua/_nav";

export const deviceDriversNav: PageNavNode = {
  key: "device-drivers",
  name: "Device Drivers",
  children: [
    {
      key: "/reference/device-drivers/get-started",
      href: "/reference/device-drivers/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/device-drivers/standalone",
      href: "/reference/device-drivers/standalone",
      name: "Standalone Driver",
    },
    {
      key: "/reference/device-drivers/timing",
      href: "/reference/device-drivers/timing",
      name: "Timing",
    },
    labjackNav,
    niNav,
    opcuaNav,
  ],
};
