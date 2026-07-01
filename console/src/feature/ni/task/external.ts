// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ANALOG_READ_LAYOUT,
  AnalogRead,
  AnalogReadSelectable,
} from "@/feature/ni/task/AnalogRead";
import {
  ANALOG_WRITE_LAYOUT,
  AnalogWrite,
  AnalogWriteSelectable,
} from "@/feature/ni/task/AnalogWrite";
import {
  COUNTER_READ_LAYOUT,
  CounterRead,
  CounterReadSelectable,
} from "@/feature/ni/task/CounterRead";
import {
  DIGITAL_READ_LAYOUT,
  DigitalRead,
  DigitalReadSelectable,
} from "@/feature/ni/task/DigitalRead";
import {
  DIGITAL_WRITE_LAYOUT,
  DigitalWrite,
  DigitalWriteSelectable,
} from "@/feature/ni/task/DigitalWrite";
import {
  ANALOG_READ_TYPE,
  ANALOG_WRITE_TYPE,
  analogReadConfigZ,
  analogWriteConfigZ,
  COUNTER_READ_TYPE,
  counterReadConfigZ,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
  digitalReadConfigZ,
  digitalWriteConfigZ,
} from "@/feature/ni/task/types";
import { createIngester } from "@/feature/task/createIngester";
import { extract } from "@/feature/task/export";
import { type Layout as TaskLayout } from "@/feature/task/Form";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { type Selector } from "@/platform/selector";

export * from "@/feature/ni/task/AnalogRead";
export * from "@/feature/ni/task/AnalogWrite";
export * from "@/feature/ni/task/CounterRead";
export * from "@/feature/ni/task/DigitalRead";
export * from "@/feature/ni/task/DigitalWrite";
export * from "@/feature/ni/task/palette";
export * from "@/feature/ni/task/types";

export const EXTRACTORS: Export.Extractors = {
  [ANALOG_READ_TYPE]: extract,
  [ANALOG_WRITE_TYPE]: extract,
  [COUNTER_READ_TYPE]: extract,
  [DIGITAL_READ_TYPE]: extract,
  [DIGITAL_WRITE_TYPE]: extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [ANALOG_READ_TYPE]: createIngester(analogReadConfigZ, ANALOG_READ_LAYOUT),
  [ANALOG_WRITE_TYPE]: createIngester(analogWriteConfigZ, ANALOG_WRITE_LAYOUT),
  [COUNTER_READ_TYPE]: createIngester(counterReadConfigZ, COUNTER_READ_LAYOUT),
  [DIGITAL_READ_TYPE]: createIngester(digitalReadConfigZ, DIGITAL_READ_LAYOUT),
  [DIGITAL_WRITE_TYPE]: createIngester(digitalWriteConfigZ, DIGITAL_WRITE_LAYOUT),
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [ANALOG_READ_TYPE]: AnalogRead,
  [ANALOG_WRITE_TYPE]: AnalogWrite,
  [COUNTER_READ_TYPE]: CounterRead,
  [DIGITAL_READ_TYPE]: DigitalRead,
  [DIGITAL_WRITE_TYPE]: DigitalWrite,
};

export const SELECTABLES: Selector.Selectable[] = [
  AnalogReadSelectable,
  AnalogWriteSelectable,
  CounterReadSelectable,
  DigitalReadSelectable,
  DigitalWriteSelectable,
];

export const ZERO_LAYOUTS: Record<string, TaskLayout> = {
  [ANALOG_READ_TYPE]: ANALOG_READ_LAYOUT,
  [ANALOG_WRITE_TYPE]: ANALOG_WRITE_LAYOUT,
  [COUNTER_READ_TYPE]: COUNTER_READ_LAYOUT,
  [DIGITAL_READ_TYPE]: DIGITAL_READ_LAYOUT,
  [DIGITAL_WRITE_TYPE]: DIGITAL_WRITE_LAYOUT,
};
