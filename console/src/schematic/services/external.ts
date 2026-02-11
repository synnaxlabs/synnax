// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Import } from "@/import";
import { Schematic } from "@/schematic";
import { ingest } from "@/schematic/services/import";

export * from "@/schematic/services/Icon";
export * from "@/schematic/services/import";
export * from "@/schematic/services/link";
export * from "@/schematic/services/ontology";
export * from "@/schematic/services/palette";

export const FILE_INGESTERS: Import.FileIngesters = { [Schematic.LAYOUT_TYPE]: ingest };
