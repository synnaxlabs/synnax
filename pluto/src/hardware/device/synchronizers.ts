// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";

import { Query } from "@/query";

export const useSetSynchronizer = (onSet: (device: device.Device) => void): void =>
  Query.useParsedListener(device.SET_CHANNEL_NAME, device.deviceZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: device.Key) => void): void =>
  Query.useParsedListener(device.DELETE_CHANNEL_NAME, device.keyZ, onDelete);

export const useStateSynchronizer = (
  onStateChange: (state: device.Status) => void,
): void =>
  Query.useParsedListener(device.STATE_CHANNEL_NAME, device.statusZ, onStateChange);
