// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { LAYOUT } from "@/hardware/task/sequence/Sequence";
import { configZ } from "@/hardware/task/sequence/types";
import { Import } from "@/import";

export const ingest = Common.Task.createIngestor(configZ, LAYOUT);

export const import_ = Import.createImporter(ingest, "control sequence");
