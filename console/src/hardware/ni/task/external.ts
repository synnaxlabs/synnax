// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ANALOG_READ_SELECTABLE, AnalogRead } from "@/hardware/ni/task/AnalogRead";
import { ANALOG_WRITE_SELECTABLE, AnalogWrite } from "@/hardware/ni/task/AnalogWrite";
import { DIGITAL_READ_SELECTABLE, DigitalRead } from "@/hardware/ni/task/DigitalRead";
import {
  DIGITAL_WRITE_SELECTABLE,
  DigitalWrite,
} from "@/hardware/ni/task/DigitalWrite";
import {
  ANALOG_READ_TYPE,
  ANALOG_WRITE_TYPE,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export * from "@/hardware/ni/task/AnalogRead";
export * from "@/hardware/ni/task/AnalogWrite";
export * from "@/hardware/ni/task/DigitalRead";
export * from "@/hardware/ni/task/DigitalWrite";
export * from "@/hardware/ni/task/palette";
export * from "@/hardware/ni/task/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [ANALOG_READ_TYPE]: AnalogRead,
  [ANALOG_WRITE_TYPE]: AnalogWrite,
  [DIGITAL_READ_TYPE]: DigitalRead,
  [DIGITAL_WRITE_TYPE]: DigitalWrite,
};

export const SELECTABLES: Layout.Selectable[] = [
  ANALOG_READ_SELECTABLE,
  ANALOG_WRITE_SELECTABLE,
  DIGITAL_READ_SELECTABLE,
  DIGITAL_WRITE_SELECTABLE,
];
