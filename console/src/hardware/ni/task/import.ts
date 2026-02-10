// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { ANALOG_READ_LAYOUT } from "@/hardware/ni/task/AnalogRead";
import { ANALOG_WRITE_LAYOUT } from "@/hardware/ni/task/AnalogWrite";
import { COUNTER_READ_LAYOUT } from "@/hardware/ni/task/CounterRead";
import { DIGITAL_READ_LAYOUT } from "@/hardware/ni/task/DigitalRead";
import { DIGITAL_WRITE_LAYOUT } from "@/hardware/ni/task/DigitalWrite";
import {
  analogReadConfigZ,
  analogWriteConfigZ,
  counterReadConfigZ,
  digitalReadConfigZ,
  digitalWriteConfigZ,
} from "@/hardware/ni/task/types";

export const ingestAnalogRead = Common.Task.createIngester(
  analogReadConfigZ,
  ANALOG_READ_LAYOUT,
);

export const ingestAnalogWrite = Common.Task.createIngester(
  analogWriteConfigZ,
  ANALOG_WRITE_LAYOUT,
);

export const ingestCounterRead = Common.Task.createIngester(
  counterReadConfigZ,
  COUNTER_READ_LAYOUT,
);

export const ingestDigitalRead = Common.Task.createIngester(
  digitalReadConfigZ,
  DIGITAL_READ_LAYOUT,
);

export const ingestDigitalWrite = Common.Task.createIngester(
  digitalWriteConfigZ,
  DIGITAL_WRITE_LAYOUT,
);
