// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device as PDevice } from "@synnaxlabs/pluto";

import { type Device, SCHEMAS } from "@/feature/ni/device/types";
import { Device as PlatformDevice } from "@/platform/device";

export const { use, useResult } = PDevice.createRetrieve(SCHEMAS);

export const useFromConfig = PlatformDevice.createUseFromConfig(useResult);

const { useResult: useResultMultiple } = PDevice.createRetrieveMultiple(SCHEMAS);

/** The devices the keys name, or undefined until they resolve or on failure. */
export const useByKeys = (keys: device.Key[]): Device[] | undefined =>
  useResultMultiple(keys.length === 0 ? null : { keys }).data;
