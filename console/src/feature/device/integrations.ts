// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentType } from "react";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { Kafka } from "@/feature/kafka";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { MQTT } from "@/feature/mqtt";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { Sift } from "@/feature/sift";
import { type Device as PlatformDevice } from "@/platform/device";
import { type Task } from "@/platform/task";

/**
 * The menu items of an integration. Its device items call its tasks, whose forms use
 * its descriptor, so the descriptor cannot hold them.
 */
interface Menus {
  /** The items it adds to the context menu of one of its devices. */
  ContextMenuItems: ComponentType<PlatformDevice.ContextMenuItemsProps>;
  /** The items it adds to the context menu of a rack whose Driver runs it. */
  RackMenuItems?: ComponentType<PlatformDevice.RackMenuItemsProps>;
}

export interface Network extends PlatformDevice.Network, Menus {}

export interface Hardware extends PlatformDevice.Hardware, Menus {
  useCreateRead: Task.UseCreate;
  /** Creates a write task, for an integration whose devices can have outputs. */
  useCreateWrite?: Task.UseCreate;
}

export type Integration = Network | Hardware;

export const NETWORK: Network[] = [
  {
    ...OPCUA.Device.NETWORK,
    ContextMenuItems: OPCUA.Device.ContextMenuItems,
    RackMenuItems: OPCUA.Device.RackMenuItems,
  },
  { ...Modbus.Device.NETWORK, ContextMenuItems: Modbus.Device.ContextMenuItems },
  { ...HTTP.Device.NETWORK, ContextMenuItems: HTTP.Device.ContextMenuItems },
  { ...Kafka.Device.NETWORK, ContextMenuItems: Kafka.Device.ContextMenuItems },
  { ...MQTT.Device.NETWORK, ContextMenuItems: MQTT.Device.ContextMenuItems },
  { ...Sift.Device.NETWORK, ContextMenuItems: Sift.Device.ContextMenuItems },
];

export const HARDWARE: Hardware[] = [
  {
    ...NI.Device.HARDWARE,
    ContextMenuItems: NI.Device.ContextMenuItems,
    RackMenuItems: NI.Device.RackMenuItems,
    useCreateRead: NI.Task.useCreateAnalogRead,
  },
  {
    ...LabJack.Device.HARDWARE,
    ContextMenuItems: LabJack.Device.ContextMenuItems,
    useCreateRead: LabJack.Task.useCreateRead,
  },
  {
    ...EtherCAT.Device.HARDWARE,
    ContextMenuItems: EtherCAT.Device.ContextMenuItems,
    useCreateRead: EtherCAT.Task.useCreateRead,
    useCreateWrite: EtherCAT.Task.useCreateWrite,
  },
];

export const INTEGRATIONS: Integration[] = [...NETWORK, ...HARDWARE];
