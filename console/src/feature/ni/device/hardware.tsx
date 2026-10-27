// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { SCAN_TYPE } from "@/feature/ni/device/scan";
import { INTEGRATION, MAKE, ZERO_PROPERTIES } from "@/feature/ni/device/types";
import { type Device } from "@/platform/device";

export const HARDWARE: Device.Hardware = {
  key: INTEGRATION,
  name: "NI",
  title: "NI hardware",
  description: "DAQ cards, cDAQ chassis, and other NI-DAQmx devices.",
  icon: <Icon.Logo.NI />,
  make: MAKE,
  scanType: SCAN_TYPE,
  zeroProperties: ZERO_PROPERTIES,
  checks: [
    "NI-DAQmx is installed on that computer. NI hardware needs Windows or Linux.",
    "The device shows in NI MAX, and no other program reserves it.",
  ],
};
