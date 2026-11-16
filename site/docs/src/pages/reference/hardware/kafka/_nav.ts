// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const KAFKA_NAV: Nav.PageNavNode = {
  key: "kafka",
  name: "Kafka",
  children: [
    {
      href: "/reference/hardware/kafka/connect-cluster",
      name: "Connect to a cluster",
    },
    {
      href: "/reference/hardware/kafka/read-task",
      name: "Read task",
    },
    {
      href: "/reference/hardware/kafka/write-task",
      name: "Write task",
    },
  ],
};
