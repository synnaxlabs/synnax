// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status as xStatus } from "@synnaxlabs/x";
import { z } from "zod";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export type Params = Key | Key[];

// The Status type combines a name with the base status from x/status
export const statusZ = xStatus.statusZ();

export interface Status extends z.infer<typeof statusZ> {
  name: string;
}

export const SET_CHANNEL_NAME = "sy_status_set";
export const DELETE_CHANNEL_NAME = "sy_status_delete";
