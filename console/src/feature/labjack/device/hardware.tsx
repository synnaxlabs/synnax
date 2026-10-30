// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { INTEGRATION, MAKE, ZERO_PROPERTIES } from "@/feature/labjack/device/types";
import { SCAN_TYPE } from "@/feature/labjack/task/types";
import { type Device } from "@/platform/device";

export const HARDWARE: Device.Hardware = {
  key: INTEGRATION,
  name: "LabJack",
  title: "LabJack hardware",
  description: "T4, T7, and T8 devices over USB or Ethernet.",
  icon: <Icon.Logo.LabJack />,
  make: MAKE,
  scanType: SCAN_TYPE,
  zeroProperties: ZERO_PROPERTIES,
  checks: [
    "LJM is installed on that computer. LabJack hardware needs Windows or macOS.",
    "Kipling and other LabJack programs are closed, because they hold the device.",
  ],
};
