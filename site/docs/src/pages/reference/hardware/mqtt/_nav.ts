// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const MQTT_NAV: PageNavNode = {
  key: "mqtt",
  name: "MQTT",
  children: [
    {
      href: "/reference/hardware/mqtt/connect-broker",
      name: "Connect to a broker",
    },
    {
      href: "/reference/hardware/mqtt/read-task",
      name: "Read task",
    },
    {
      href: "/reference/hardware/mqtt/write-task",
      name: "Write task",
    },
    {
      href: "/reference/hardware/mqtt/sparkplug",
      name: "Sparkplug B",
    },
  ],
};
