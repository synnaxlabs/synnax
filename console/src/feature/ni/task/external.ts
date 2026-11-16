// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AnalogRead } from "@/feature/ni/task/AnalogRead";
import { AnalogWrite } from "@/feature/ni/task/AnalogWrite";
import { CounterRead } from "@/feature/ni/task/CounterRead";
import { DigitalRead } from "@/feature/ni/task/DigitalRead";
import { DigitalWrite } from "@/feature/ni/task/DigitalWrite";
import {
  ANALOG_READ_TYPE,
  ANALOG_WRITE_TYPE,
  COUNTER_READ_TYPE,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
} from "@/feature/ni/task/types";
import { type Task } from "@/platform/task";

export * from "@/feature/ni/task/AnalogRead";
export * from "@/feature/ni/task/AnalogWrite";
export * from "@/feature/ni/task/commands";
export * from "@/feature/ni/task/CounterRead";
export * from "@/feature/ni/task/DigitalRead";
export * from "@/feature/ni/task/DigitalWrite";
export * from "@/feature/ni/task/integration";
export * from "@/feature/ni/task/types";
export * from "@/feature/ni/task/useToggleScanner";

export const FORMS: Task.Forms = {
  [ANALOG_READ_TYPE]: AnalogRead,
  [ANALOG_WRITE_TYPE]: AnalogWrite,
  [COUNTER_READ_TYPE]: CounterRead,
  [DIGITAL_READ_TYPE]: DigitalRead,
  [DIGITAL_WRITE_TYPE]: DigitalWrite,
};
