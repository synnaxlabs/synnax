// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const OPC_UA_NAV: PageNavNode = {
  key: "opc-ua",
  name: "OPC UA",
  children: [
    {
      key: "/reference/device-drivers/opc-ua/connect-server",
      href: "/reference/device-drivers/opc-ua/connect-server",
      name: "Connect to a Server",
    },
    {
      key: "/reference/device-drivers/opc-ua/read-task",
      href: "/reference/device-drivers/opc-ua/read-task",
      name: "Read Task",
    },
    {
      key: "/reference/device-drivers/opc-ua/write-task",
      href: "/reference/device-drivers/opc-ua/write-task",
      name: "Write Task",
    },
  ],
};
