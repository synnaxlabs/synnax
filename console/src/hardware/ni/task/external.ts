// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Common } from "@/hardware/common";
import { ANALOG_READ_SELECTABLE, AnalogRead } from "@/hardware/ni/task/AnalogRead";
import { ANALOG_WRITE_SELECTABLE, AnalogWrite } from "@/hardware/ni/task/AnalogWrite";
import { COUNTER_READ_SELECTABLE, CounterRead } from "@/hardware/ni/task/CounterRead";
import {
  COUNTER_WRITE_SELECTABLE,
  CounterWrite,
} from "@/hardware/ni/task/CounterWrite";
import { DIGITAL_READ_SELECTABLE, DigitalRead } from "@/hardware/ni/task/DigitalRead";
import {
  DIGITAL_WRITE_SELECTABLE,
  DigitalWrite,
} from "@/hardware/ni/task/DigitalWrite";
import {
  ingestAnalogRead,
  ingestAnalogWrite,
  ingestCounterRead,
  ingestCounterWrite,
  ingestDigitalRead,
  ingestDigitalWrite,
} from "@/hardware/ni/task/import";
import {
  ANALOG_READ_TYPE,
  ANALOG_WRITE_TYPE,
  COUNTER_READ_TYPE,
  COUNTER_WRITE_TYPE,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
} from "@/hardware/ni/task/types";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/ni/task/AnalogRead";
export * from "@/hardware/ni/task/AnalogWrite";
export * from "@/hardware/ni/task/CounterRead";
export * from "@/hardware/ni/task/CounterWrite";
export * from "@/hardware/ni/task/DigitalRead";
export * from "@/hardware/ni/task/DigitalWrite";
export * from "@/hardware/ni/task/palette";
export * from "@/hardware/ni/task/types";

export const EXTRACTORS: Export.Extractors = {
  [ANALOG_READ_TYPE]: Common.Task.extract,
  [ANALOG_WRITE_TYPE]: Common.Task.extract,
  [COUNTER_READ_TYPE]: Common.Task.extract,
  [COUNTER_WRITE_TYPE]: Common.Task.extract,
  [DIGITAL_READ_TYPE]: Common.Task.extract,
  [DIGITAL_WRITE_TYPE]: Common.Task.extract,
};

export const FILE_INGESTORS: Import.FileIngestors = {
  [ANALOG_READ_TYPE]: ingestAnalogRead,
  [ANALOG_WRITE_TYPE]: ingestAnalogWrite,
  [COUNTER_READ_TYPE]: ingestCounterRead,
  [COUNTER_WRITE_TYPE]: ingestCounterWrite,
  [DIGITAL_READ_TYPE]: ingestDigitalRead,
  [DIGITAL_WRITE_TYPE]: ingestDigitalWrite,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [ANALOG_READ_TYPE]: AnalogRead,
  [ANALOG_WRITE_TYPE]: AnalogWrite,
  [COUNTER_READ_TYPE]: CounterRead,
  [COUNTER_WRITE_TYPE]: CounterWrite,
  [DIGITAL_READ_TYPE]: DigitalRead,
  [DIGITAL_WRITE_TYPE]: DigitalWrite,
};

export const SELECTABLES: Selector.Selectable[] = [
  ANALOG_READ_SELECTABLE,
  ANALOG_WRITE_SELECTABLE,
  COUNTER_READ_SELECTABLE,
  COUNTER_WRITE_SELECTABLE,
  DIGITAL_READ_SELECTABLE,
  DIGITAL_WRITE_SELECTABLE,
];
