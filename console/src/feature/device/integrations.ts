// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { type Device as PlatformDevice } from "@/platform/device";
import { type Task } from "@/platform/task";

export interface Hardware extends PlatformDevice.Hardware {
  useCreateRead: Task.UseCreate;
}

export const NETWORK: PlatformDevice.Network[] = [
  OPCUA.Device.NETWORK,
  Modbus.Device.NETWORK,
  HTTP.Device.NETWORK,
];

export const HARDWARE: Hardware[] = [
  { ...NI.Device.HARDWARE, useCreateRead: NI.Task.useCreateAnalogRead },
  { ...LabJack.Device.HARDWARE, useCreateRead: LabJack.Task.useCreateRead },
  { ...EtherCAT.Device.HARDWARE, useCreateRead: EtherCAT.Task.useCreateRead },
];

export const INTEGRATIONS: PlatformDevice.Integration[] = [...NETWORK, ...HARDWARE];
