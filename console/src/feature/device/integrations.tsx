// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type record } from "@synnaxlabs/x";
import { type ComponentType } from "react";

import { type Make } from "@/feature/device/make";
import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { type Device as PlatformDevice } from "@/platform/device";
import { type Task } from "@/platform/task";

export interface Integration {
  /** The name a rack lists in its integrations when its Driver runs this one. */
  key: string;
  /** The display name, such as "OPC UA". */
  name: string;
  title: string;
  description: string;
  icon: Icon.ReactElement;
}

/** An integration the user connects to by address. */
export interface Network extends Integration {
  Connect: ComponentType<PlatformDevice.ConnectProps>;
}

/** An integration whose hardware the Driver finds by scanning. */
export interface Hardware extends Integration {
  make: Make;
  /** The type of the scan task each Driver runs for this integration. */
  scanType: string;
  /** What to check when the Driver does not find the hardware. */
  checks: string[];
  /** The properties a device of this make starts with when the user configures it. */
  zeroProperties: record.Unknown;
  useCreateRead: Task.UseCreate;
}

export const NETWORK: Network[] = [
  {
    key: OPCUA.Device.INTEGRATION,
    name: "OPC UA",
    title: "OPC UA server",
    description: "Read and write nodes on an OPC UA server.",
    icon: <Icon.Logo.OPCUA />,
    Connect: OPCUA.Device.Connect,
  },
  {
    key: Modbus.Device.INTEGRATION,
    name: "Modbus",
    title: "Modbus TCP server",
    description: "Read and write registers and coils on a Modbus TCP server.",
    icon: <Icon.Logo.Modbus />,
    Connect: Modbus.Device.Connect,
  },
  {
    key: HTTP.Device.INTEGRATION,
    name: "HTTP",
    title: "HTTP server",
    description: "Poll and send JSON to an HTTP API.",
    icon: <Icon.Logo.HTTP />,
    Connect: HTTP.Device.Connect,
  },
];

const DRIVER_CHECK = "The Synnax Driver runs on the computer the hardware connects to.";

export const HARDWARE: Hardware[] = [
  {
    key: NI.Device.INTEGRATION,
    name: "NI",
    title: "NI hardware",
    description: "DAQ cards, cDAQ chassis, and other NI-DAQmx devices.",
    icon: <Icon.Logo.NI />,
    make: NI.Device.MAKE,
    scanType: NI.Task.SCAN_TYPE,
    zeroProperties: NI.Device.ZERO_PROPERTIES,
    useCreateRead: NI.Task.useCreateAnalogRead,
    checks: [
      DRIVER_CHECK,
      "NI-DAQmx is installed on that computer. NI hardware needs Windows or Linux.",
      "The device shows in NI MAX, and no other program reserves it.",
    ],
  },
  {
    key: LabJack.Device.INTEGRATION,
    name: "LabJack",
    title: "LabJack hardware",
    description: "T4, T7, and T8 devices over USB or Ethernet.",
    icon: <Icon.Logo.LabJack />,
    make: LabJack.Device.MAKE,
    scanType: LabJack.Task.SCAN_TYPE,
    zeroProperties: LabJack.Device.ZERO_PROPERTIES,
    useCreateRead: LabJack.Task.useCreateRead,
    checks: [
      DRIVER_CHECK,
      "The LabJack LJM library is installed on that computer.",
      "Kipling and other LabJack programs are closed, because they hold the device.",
    ],
  },
  {
    key: EtherCAT.Device.INTEGRATION,
    name: "EtherCAT",
    title: "EtherCAT network",
    description: "Slaves on an EtherCAT network adapter.",
    icon: <Icon.Logo.EtherCAT />,
    make: EtherCAT.Device.MAKE,
    scanType: EtherCAT.Task.SCAN_TYPE,
    zeroProperties: EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
    useCreateRead: EtherCAT.Task.useCreateRead,
    checks: [
      DRIVER_CHECK,
      "The Driver can open the adapter. On Linux, this needs root or CAP_NET_RAW.",
      "The slaves have power and a cable to that adapter.",
    ],
  },
];
