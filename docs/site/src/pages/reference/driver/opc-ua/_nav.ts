// Copyright 2026 Synnax Labs, Inc.
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
      key: "/reference/driver/opc-ua/get-started",
      href: "/reference/driver/opc-ua/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/opc-ua/connect-server",
      href: "/reference/driver/opc-ua/connect-server",
      name: "Connect to a Server",
    },
    {
      key: "/reference/driver/opc-ua/read-task",
      href: "/reference/driver/opc-ua/read-task",
      name: "Read Task",
    },
    {
      key: "/reference/driver/opc-ua/write-task",
      href: "/reference/driver/opc-ua/write-task",
      name: "Write Task",
    },
  ],
};
