// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const MODBUS_TCP_NAV: PageNavNode = {
  key: "modbus-tcp",
  name: "Modbus TCP",
  children: [
    {
      key: "/reference/device-drivers/modbus-tcp/connect-device",
      href: "/reference/device-drivers/modbus-tcp/connect-device",
      name: "Connect to a Server",
    },
    {
      key: "/reference/device-drivers/modbus-tcp/read-task",
      href: "/reference/device-drivers/modbus-tcp/read-task",
      name: "Configure a Read Task",
    },
    {
      key: "/reference/device-drivers/modbus-tcp/write-task",
      href: "/reference/device-drivers/modbus-tcp/write-task",
      name: "Configure a Write Task",
    },
  ],
};
