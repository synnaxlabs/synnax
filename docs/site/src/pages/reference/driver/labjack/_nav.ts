// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const LABJACK_NAV: PageNavNode = {
  key: "labjack",
  name: "LabJack",
  children: [
    {
      key: "/reference/driver/labjack/get-started",
      href: "/reference/driver/labjack/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/driver/labjack/configure-device",
      href: "/reference/driver/labjack/configure-device",
      name: "Configure a Device",
    },
    {
      key: "/reference/driver/labjack/read-task",
      href: "/reference/driver/labjack/read-task",
      name: "Read Task",
    },
    {
      key: "/reference/driver/labjack/write-task",
      href: "/reference/driver/labjack/write-task",
      name: "Write Task",
    },
  ],
};
