// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { SCAN_TYPE } from "@/feature/ethercat/device/scan";
import {
  INTEGRATION,
  MAKE,
  ZERO_SLAVE_PROPERTIES,
} from "@/feature/ethercat/device/types";
import { type Device } from "@/platform/device";

export const HARDWARE: Device.Hardware = {
  key: INTEGRATION,
  name: "EtherCAT",
  title: "EtherCAT network",
  description: "Slaves on an EtherCAT network adapter.",
  icon: <Icon.Logo.EtherCAT />,
  make: MAKE,
  scanType: SCAN_TYPE,
  zeroProperties: ZERO_SLAVE_PROPERTIES,
  checks: [
    "The Driver can open the adapter. On Linux, this needs root or CAP_NET_RAW.",
    "The slaves have power and a cable to that adapter.",
  ],
};
