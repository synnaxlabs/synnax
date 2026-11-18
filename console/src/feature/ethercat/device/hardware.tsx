// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { Configure } from "@/feature/ethercat/device/Configure";
import { formatLocation, slotZ } from "@/feature/ethercat/device/location";
import { ScanAction } from "@/feature/ethercat/device/ScanAction";
import { INTEGRATION_KEY, MAKE, SLAVE_MODEL } from "@/feature/ethercat/device/types";
import { SCAN_TYPE } from "@/feature/ethercat/task/types";
import { type Device } from "@/platform/device";

export const HARDWARE: Device.Hardware = {
  key: INTEGRATION_KEY,
  name: "EtherCAT",
  title: "EtherCAT network",
  description: "Modules on an EtherCAT network adapter.",
  icon: <Icon.Logo.EtherCAT />,
  make: MAKE,
  scanType: SCAN_TYPE,
  location: { shape: "slot" },
  Configure,
  ScanAction,
  modelNames: { [SLAVE_MODEL]: "Module" },
  checks: [
    "The Driver can open the adapter. On Windows, this needs Npcap, installed with " +
      "the default options. On Linux, it needs root or CAP_NET_RAW.",
    "The modules have power and a cable to that adapter.",
  ],
  formatLocation: (device) =>
    formatLocation(device.location, slotZ.parse(device.properties)),
};
