// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Import } from "@/import";
import { Table } from "@/table";
import { ingest } from "@/table/services/import";

export * from "@/table/services/Icon";
export * from "@/table/services/import";
export * from "@/table/services/link";
export * from "@/table/services/ontology";
export * from "@/table/services/palette";

export const FILE_INGESTERS: Import.FileIngesters = { [Table.LAYOUT_TYPE]: ingest };
