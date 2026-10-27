// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const ETHERCAT_NAV: PageNavNode = {
  key: "ethercat",
  name: "EtherCAT",
  children: [
    {
      href: "/reference/hardware/ethercat/configure-device",
      name: "Configure a device",
    },
    {
      href: "/reference/hardware/ethercat/read-task",
      name: "Read task",
    },
    {
      href: "/reference/hardware/ethercat/write-task",
      name: "Write task",
    },
  ],
};
