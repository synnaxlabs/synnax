// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/component/layout";
import { type Selector } from "@/component/selector";
import { type Export } from "@/service/export";
import { type Import } from "@/service/import";
import {
  ANALOG_READ_LAYOUT,
  AnalogRead,
  AnalogReadSelectable,
} from "@/service/ni/task/AnalogRead";
import {
  ANALOG_WRITE_LAYOUT,
  AnalogWrite,
  AnalogWriteSelectable,
} from "@/service/ni/task/AnalogWrite";
import {
  COUNTER_READ_LAYOUT,
  CounterRead,
  CounterReadSelectable,
} from "@/service/ni/task/CounterRead";
import {
  DIGITAL_READ_LAYOUT,
  DigitalRead,
  DigitalReadSelectable,
} from "@/service/ni/task/DigitalRead";
import {
  DIGITAL_WRITE_LAYOUT,
  DigitalWrite,
  DigitalWriteSelectable,
} from "@/service/ni/task/DigitalWrite";
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
} from "@/service/ni/task/types";
import { createIngester } from "@/service/task/createIngester";
import { extract } from "@/service/task/export";
import { type Layout as TaskLayout } from "@/service/task/Form";

export * from "@/service/ni/task/AnalogRead";
export * from "@/service/ni/task/AnalogWrite";
export * from "@/service/ni/task/CounterRead";
export * from "@/service/ni/task/DigitalRead";
export * from "@/service/ni/task/DigitalWrite";
export * from "@/service/ni/task/palette";
export * from "@/service/ni/task/types";

export const EXTRACTORS: Export.Extractors = {
  [ANALOG_READ_TYPE]: extract,
  [ANALOG_WRITE_TYPE]: extract,
  [COUNTER_READ_TYPE]: extract,
  [DIGITAL_READ_TYPE]: extract,
  [DIGITAL_WRITE_TYPE]: extract,
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [ANALOG_READ_TYPE]: createIngester(analogReadConfigZ, ANALOG_READ_LAYOUT),
  [ANALOG_WRITE_TYPE]: createIngester(
    analogWriteConfigZ,
    ANALOG_WRITE_LAYOUT,
  ),
  [COUNTER_READ_TYPE]: createIngester(
    counterReadConfigZ,
    COUNTER_READ_LAYOUT,
  ),
  [DIGITAL_READ_TYPE]: createIngester(
    digitalReadConfigZ,
    DIGITAL_READ_LAYOUT,
  ),
  [DIGITAL_WRITE_TYPE]: createIngester(
    digitalWriteConfigZ,
    DIGITAL_WRITE_LAYOUT,
  ),
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
