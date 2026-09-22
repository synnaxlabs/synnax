// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const MQTT_NAV: PageNavNode = {
  key: "mqtt",
  name: "MQTT",
  children: [
    {
      key: "/reference/driver/mqtt/connect-broker",
      href: "/reference/driver/mqtt/connect-broker",
      name: "Connect to a Broker",
    },
    {
      key: "/reference/driver/mqtt/read-task",
      href: "/reference/driver/mqtt/read-task",
      name: "Read Task",
    },
    {
      key: "/reference/driver/mqtt/write-task",
      href: "/reference/driver/mqtt/write-task",
      name: "Write Task",
    },
    {
      key: "/reference/driver/mqtt/sparkplug",
      href: "/reference/driver/mqtt/sparkplug",
      name: "Sparkplug B",
    },
  ],
};
