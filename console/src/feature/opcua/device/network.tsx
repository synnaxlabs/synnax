// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { SCAN_TYPE } from "@/feature/opcua/device/scan";
import { INTEGRATION, MAKE } from "@/feature/opcua/device/types";
import { Connect } from "@/feature/opcua/device/useConnectModal";
import { type Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION,
  name: "OPC UA",
  title: "OPC UA server",
  description: "Read and write nodes on an OPC UA server.",
  icon: <Icon.Logo.OPCUA />,
  make: MAKE,
  scanType: SCAN_TYPE,
  Connect,
};
