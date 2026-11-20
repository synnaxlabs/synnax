// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ANALOG_READ_TYPE,
  ANALOG_WRITE_TYPE,
  COUNTER_READ_TYPE,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
} from "@/feature/ni/task/types";
import { Panel } from "@/platform/panel";
import { type Task } from "@/platform/task";

export * from "@/feature/ni/task/commands";
export * from "@/feature/ni/task/create";
export * from "@/feature/ni/task/integration";
export * from "@/feature/ni/task/types";
export * from "@/feature/ni/task/useToggleScanner";

export const FORMS: Task.Forms = {
  [ANALOG_READ_TYPE]: Panel.lazyComponent(
    () => import("@/feature/ni/task/AnalogRead"),
    "AnalogRead",
  ),
  [ANALOG_WRITE_TYPE]: Panel.lazyComponent(
    () => import("@/feature/ni/task/AnalogWrite"),
    "AnalogWrite",
  ),
  [COUNTER_READ_TYPE]: Panel.lazyComponent(
    () => import("@/feature/ni/task/CounterRead"),
    "CounterRead",
  ),
  [DIGITAL_READ_TYPE]: Panel.lazyComponent(
    () => import("@/feature/ni/task/DigitalRead"),
    "DigitalRead",
  ),
  [DIGITAL_WRITE_TYPE]: Panel.lazyComponent(
    () => import("@/feature/ni/task/DigitalWrite"),
    "DigitalWrite",
  ),
};
