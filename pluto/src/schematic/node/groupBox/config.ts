// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

export const VARIANT = "group_box" as const;

/** PADDING is the gap between a group box's edge and its members' bounds. */
export const PADDING = 20;

/** TOP_PADDING is the larger gap above the members' bounds. */
export const TOP_PADDING = 40;

export type Config = schematic.GroupBoxNodeConfig;
