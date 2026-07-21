// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { type Device, deviceZ, type Key, keyZ } from "@/device/types.gen";

export const SET_CHANNEL_NAME = "sy_device_set";
export const DELETE_CHANNEL_NAME = "sy_device_delete";

export const STORE_KEY = "devices";

const genericDeviceZ = deviceZ();

/** Registers the device table on the given cache. */
export const bindStore = (engine: cache.Cache): void => {
  // Explicitly omit 'status' from the device type to make sure we aren't storing two
  // copies of the statuses in the store.
  const table = () => engine.table<Key, Omit<Device, "status">>(STORE_KEY);
  const set: cache.ChannelListener<{}, typeof genericDeviceZ> = {
    channel: SET_CHANNEL_NAME,
    schema: genericDeviceZ,
    onChange: ({ changed: { status: _, ...device } }) =>
      table().set(device.key, device),
  };
  const del: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => table().delete(changed),
  };
  engine.registerTable<Key, Omit<Device, "status">>(STORE_KEY, {
    listeners: [set, del],
  });
};
