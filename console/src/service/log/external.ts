// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/component/layout";
import { LAYOUT_TYPE } from "@/component/log/layout";
import { type Selector } from "@/component/selector";
import { type Export } from "@/service/export";
import { extract } from "@/service/log/export";
import { FILE_INGESTERS } from "@/service/log/import";
import { Log } from "@/service/log/Log";
import { Selectable } from "@/service/log/Selectable";

export * from "@/component/log/layout";
export * from "@/component/log/useCreate";
export * from "@/service/log/link";
export * from "@/service/log/ontology";
export * from "@/service/log/palette";
export * from "@/service/log/Selectable";
export * from "@/service/log/toolbar";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const ImEx = { FILE_INGESTERS };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Log };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
