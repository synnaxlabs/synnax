// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { READ_LAYOUT } from "@/hardware/modbus/task/Read";
import { readConfigZ, writeConfigZ } from "@/hardware/modbus/task/types";
import { WRITE_LAYOUT } from "@/hardware/modbus/task/Write";
import { Import } from "@/import";

export const ingestRead = Common.Task.createIngestor(readConfigZ, READ_LAYOUT);

export const importRead = Import.createImporter(ingestRead, "Modbus Read Task");

export const ingestWrite = Common.Task.createIngestor(writeConfigZ, WRITE_LAYOUT);

export const importWrite = Import.createImporter(ingestWrite, "Modbus Write Task");
