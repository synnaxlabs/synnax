// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ANALOG_READ_SELECTABLE,
  ConfigureAnalogRead,
} from "@/hardware/ni/task/AnalogRead";
import {
  ConfigureDigitalRead,
  DIGITAL_READ_SELECTABLE,
} from "@/hardware/ni/task/DigitalRead";
import {
  ConfigureDigitalWrite,
  DIGITAL_WRITE_SELECTABLE,
} from "@/hardware/ni/task/DigitalWrite";
import {
  ANALOG_READ_TYPE,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
} from "@/hardware/ni/task/migrations";
import { Layout } from "@/layout";

export * from "@/hardware/ni/task/AnalogRead";
export * from "@/hardware/ni/task/DigitalRead";
export * from "@/hardware/ni/task/DigitalWrite";
export * from "@/hardware/ni/task/migrations";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [ANALOG_READ_TYPE]: ConfigureAnalogRead,
  [DIGITAL_WRITE_TYPE]: ConfigureDigitalWrite,
  [DIGITAL_READ_TYPE]: ConfigureDigitalRead,
};

export const SELECTABLES: Layout.Selectable[] = [
  DIGITAL_READ_SELECTABLE,
  DIGITAL_WRITE_SELECTABLE,
  ANALOG_READ_SELECTABLE,
];
