// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { INTEGRATION_KEY, MAKE } from "@/feature/serial/device/types";
import { Connect } from "@/feature/serial/device/useConnectModal";
import type { Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION_KEY,
  name: "Serial",
  title: "Serial device",
  description: "Read and write ICD messages on a serial port.",
  icon: <Icon.Serial />,
  make: MAKE,
  location: { shape: "plain" },
  Connect,
};
