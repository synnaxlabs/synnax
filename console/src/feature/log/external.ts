// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/feature/export";
import { extract } from "@/feature/log/export";
import { FILE_INGESTERS } from "@/feature/log/import";
import { Log } from "@/feature/log/Log";
import { Selectable } from "@/feature/log/Selectable";
import { type Layout } from "@/primitive/layout";
import { LAYOUT_TYPE } from "@/primitive/log/layout";
import { type Selector } from "@/primitive/selector";

export * from "@/feature/log/link";
export * from "@/feature/log/ontology";
export * from "@/feature/log/palette";
export * from "@/feature/log/Selectable";
export * from "@/feature/log/toolbar";
export * from "@/primitive/log/layout";
export * from "@/primitive/log/useCreate";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const ImEx = { FILE_INGESTERS };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Log };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
