// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ingest } from "@/arc/import";
import { TYPE } from "@/arc/types";
import { type Import } from "@/import";

export * from "@/arc/import";
export * from "@/arc/services/ontology";
export * from "@/arc/services/palette";

export const FILE_INGESTERS: Import.FileIngesters = { [TYPE]: ingest };
