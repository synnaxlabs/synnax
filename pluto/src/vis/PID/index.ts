// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PID as CorePID, usePID } from "@/vis/pid/PID";
export type {
  PIDProps,
  UsePIDProps,
  UsePIDReturn,
  PIDNode,
  PIDEdge,
  PIDElementProps,
  PIDViewport,
} from "@/vis/pid/PID";

type CorePIDType = typeof CorePID;

interface PIDType extends CorePIDType {
  use: typeof usePID;
}

export const PID = CorePID as PIDType;

PID.use = usePID;
