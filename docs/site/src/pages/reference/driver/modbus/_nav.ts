// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const MODBUS_NAV: PageNavNode = {
  key: "modbus",
  name: "Modbus",
  children: [
    {
      key: "/reference/driver/modbus/get-started",
      href: "/reference/driver/modbus/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/modbus/connect-server",
      href: "/reference/driver/modbus/connect-server",
      name: "Connect to a Server",
    },
    {
      key: "/reference/driver/modbus/read-task",
      href: "/reference/driver/modbus/read-task",
      name: "Read Task",
    },
    {
      key: "/reference/driver/modbus/write-task",
      href: "/reference/driver/modbus/write-task",
      name: "Write Task",
    },
  ],
};
