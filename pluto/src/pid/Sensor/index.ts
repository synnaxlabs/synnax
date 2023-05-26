// Copyright 2023 Synnax Labs, Inc.
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SensorNumeric } from "@/pid/Sensor/SensorNumeric";
export type { SensorNumericProps } from "@/pid/Sensor/SensorNumeric";

interface SensorType {
  Numeric: typeof SensorNumeric;
}

export const Sensor: SensorType = {
  Numeric: SensorNumeric,
};
