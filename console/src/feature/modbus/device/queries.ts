// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device as PDevice } from "@synnaxlabs/pluto";

import { SCHEMAS } from "@/feature/modbus/device/types";
import { Device as PlatformDevice } from "@/platform/device";

export const { use, useResult } = PDevice.createRetrieve(SCHEMAS);

export const useFromConfig = PlatformDevice.createUseFromConfig(useResult);
