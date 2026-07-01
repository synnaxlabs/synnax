// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { extract } from "@/feature/log/export";
import { ingest } from "@/feature/log/import";
import { Log } from "@/feature/log/Log";
import { Selectable } from "@/feature/log/Selectable";
import { type Export } from "@/platform/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { Log as Platform } from "@/platform/log";
import { type Selector } from "@/platform/selector";

export * from "@/feature/log/import";
export * from "@/feature/log/link";
export * from "@/feature/log/ontology";
export * from "@/feature/log/palette";
export * from "@/feature/log/Selectable";
export * from "@/feature/log/toolbar";
export * from "@/platform/log/layout";

export const EXTRACTORS: Export.Extractors = { [Platform.LAYOUT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = { [Platform.LAYOUT_TYPE]: ingest };

export const LAYOUTS: Record<string, Layout.Renderer> = { [Platform.LAYOUT_TYPE]: Log };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
export * from "@/platform/log/useCreate";
