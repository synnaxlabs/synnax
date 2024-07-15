// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const opcuaNav: PageNavNode = {
  key: "opcua",
  name: "OPC UA",
  children: [
    {
      key: "/reference/device-drivers/opcua/connect-server",
      href: "/reference/device-drivers/opcua/connect-server",
      name: "Connect to a Server",
    },
    {
      key: "/reference/device-drivers/opcua/read-task",
      href: "/reference/device-drivers/opcua/read-task",
      name: "Configure a Read Task",
    },
  ],
};
