// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";
import { LABJACK_NAV } from "@/pages/reference/driver/labjack/_nav";
import { MODBUS_NAV } from "@/pages/reference/driver/modbus/_nav";
import { NI_NAV } from "@/pages/reference/driver/ni/_nav";
import { OPC_UA_NAV } from "@/pages/reference/driver/opc-ua/_nav";

export const DRIVER_NAV: PageNavNode = {
  key: "driver",
  name: "Driver",
  children: [
    {
      key: "/reference/driver/get-started",
      href: "/reference/driver/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/installation",
      href: "/reference/driver/installation",
      name: "Installation",
    },
    {
      key: "/reference/driver/timing",
      href: "/reference/driver/timing",
      name: "Timing",
    },
    {
      key: "/reference/driver/task-basics",
      href: "/reference/driver/task-basics",
      name: "Task Basics",
    },
    LABJACK_NAV,
    MODBUS_NAV,
    NI_NAV,
    OPC_UA_NAV,
  ],
};
