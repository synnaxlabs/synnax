// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";
import { ETHERCAT_NAV } from "@/pages/reference/hardware/ethercat/_nav";
import { HTTP_NAV } from "@/pages/reference/hardware/http/_nav";
import { LABJACK_NAV } from "@/pages/reference/hardware/labjack/_nav";
import { MODBUS_NAV } from "@/pages/reference/hardware/modbus/_nav";
import { NI_NAV } from "@/pages/reference/hardware/ni/_nav";
import { OPC_UA_NAV } from "@/pages/reference/hardware/opc-ua/_nav";

export const HARDWARE_NAV: Nav.PageNavNode = {
  key: "hardware",
  name: "Hardware",
  children: [
    {
      href: "/reference/hardware",
      name: "Overview",
    },
    {
      href: "/reference/hardware/devices",
      name: "Devices",
    },
    {
      href: "/reference/hardware/tasks",
      name: "Tasks",
    },
    {
      href: "/reference/hardware/start-and-stop-tasks",
      name: "Start and stop",
    },
    {
      href: "/reference/hardware/troubleshooting",
      name: "Troubleshooting",
    },
    ETHERCAT_NAV,
    HTTP_NAV,
    LABJACK_NAV,
    MODBUS_NAV,
    NI_NAV,
    OPC_UA_NAV,
    {
      href: "/reference/hardware/custom",
      name: "Custom devices",
      edition: "enterprise",
    },
  ],
};
