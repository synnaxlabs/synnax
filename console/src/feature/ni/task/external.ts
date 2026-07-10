// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AnalogRead, AnalogReadSelectable } from "@/feature/ni/task/AnalogRead";
import { AnalogWrite, AnalogWriteSelectable } from "@/feature/ni/task/AnalogWrite";
import { CounterRead, CounterReadSelectable } from "@/feature/ni/task/CounterRead";
import { DigitalRead, DigitalReadSelectable } from "@/feature/ni/task/DigitalRead";
import { DigitalWrite, DigitalWriteSelectable } from "@/feature/ni/task/DigitalWrite";
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
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export * from "@/feature/ni/task/AnalogRead";
export * from "@/feature/ni/task/AnalogWrite";
export * from "@/feature/ni/task/commands";
export * from "@/feature/ni/task/CounterRead";
export * from "@/feature/ni/task/DigitalRead";
export * from "@/feature/ni/task/DigitalWrite";
export * from "@/feature/ni/task/types";
export * from "@/feature/ni/task/useToggleScanner";

export const EXTRACTORS: Export.Extractors = {
  [ANALOG_READ_TYPE]: Task.extract,
  [ANALOG_WRITE_TYPE]: Task.extract,
  [COUNTER_READ_TYPE]: Task.extract,
  [DIGITAL_READ_TYPE]: Task.extract,
  [DIGITAL_WRITE_TYPE]: Task.extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [ANALOG_READ_TYPE]: Task.createIngester(analogReadConfigZ, ANALOG_READ_TYPE),
  [ANALOG_WRITE_TYPE]: Task.createIngester(analogWriteConfigZ, ANALOG_WRITE_TYPE),
  [COUNTER_READ_TYPE]: Task.createIngester(counterReadConfigZ, COUNTER_READ_TYPE),
  [DIGITAL_READ_TYPE]: Task.createIngester(digitalReadConfigZ, DIGITAL_READ_TYPE),
  [DIGITAL_WRITE_TYPE]: Task.createIngester(digitalWriteConfigZ, DIGITAL_WRITE_TYPE),
};

export const TABS: Panel.Tabs = {
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
