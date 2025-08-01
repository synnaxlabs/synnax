// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Import } from "@/import";
import { LinePlot } from "@/lineplot";
import { ingest } from "@/lineplot/services/import";

export * from "@/lineplot/services/Icon";
export * from "@/lineplot/services/import";
export * from "@/lineplot/services/link";
export * from "@/lineplot/services/ontology";
export * from "@/lineplot/services/palette";

export const FILE_INGESTORS: Import.FileIngestors = { [LinePlot.LAYOUT_TYPE]: ingest };
