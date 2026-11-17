// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { INTEGRATION_KEY, MAKE } from "@/feature/kafka/device/types";
import { Connect } from "@/feature/kafka/device/useConnectModal";
import { SCAN_TYPE } from "@/feature/kafka/task/types";
import { type Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION_KEY,
  name: "Kafka",
  title: "Kafka cluster",
  description: "Consume and produce JSON messages on Kafka topics.",
  icon: <Icon.Logo.Kafka />,
  make: MAKE,
  scanType: SCAN_TYPE,
  location: { shape: "list" },
  Connect,
};
