// Copyright 2025 Synnax Labs, Inc.
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
import { DIGITAL_READ_LAYOUT } from "@/hardware/ni/task/DigitalRead";
import { DIGITAL_WRITE_LAYOUT } from "@/hardware/ni/task/DigitalWrite";
import {
  analogReadConfigZ,
  analogWriteConfigZ,
  digitalReadConfigZ,
  digitalWriteConfigZ,
} from "@/hardware/ni/task/types";
import { Import } from "@/import";

export const ingestAnalogRead = Common.Task.createIngestor(
  analogReadConfigZ,
  ANALOG_READ_LAYOUT,
);

export const importAnalogRead = Import.createImporter(
  ingestAnalogRead,
  "NI analog read task",
);

export const ingestAnalogWrite = Common.Task.createIngestor(
  analogWriteConfigZ,
  ANALOG_WRITE_LAYOUT,
);

export const importAnalogWrite = Import.createImporter(
  ingestAnalogWrite,
  "NI analog write task",
);

export const ingestDigitalRead = Common.Task.createIngestor(
  digitalReadConfigZ,
  DIGITAL_READ_LAYOUT,
);

export const importDigitalRead = Import.createImporter(
  ingestDigitalRead,
  "NI digital read task",
);

export const ingestDigitalWrite = Common.Task.createIngestor(
  digitalWriteConfigZ,
  DIGITAL_WRITE_LAYOUT,
);

export const importDigitalWrite = Import.createImporter(
  ingestDigitalWrite,
  "NI digital write task",
);
