// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { INTEGRATION } from "@/feature/modbus/device/types";
import { Connect } from "@/feature/modbus/device/useConnectModal";
import { type Device } from "@/platform/device";

export const NETWORK: Device.Network = {
  key: INTEGRATION,
  name: "Modbus",
  title: "Modbus server",
  description: "Read and write registers and coils over Modbus TCP or Modbus RTU.",
  icon: <Icon.Logo.Modbus />,
  Connect,
};
