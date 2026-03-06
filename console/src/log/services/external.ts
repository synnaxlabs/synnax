// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Import } from "@/import";
import { Log } from "@/log";
import { ingest } from "@/log/services/import";

export * from "@/log/services/Icon";
export * from "@/log/services/import";
export * from "@/log/services/link";
export * from "@/log/services/ontology";
export * from "@/log/services/palette";

export const FILE_INGESTERS: Import.FileIngesters = { [Log.LAYOUT_TYPE]: ingest };
