// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Log } from "@/layered/service/log/body/Log";
import { extract } from "@/layered/service/log/imex/export";
import { LAYOUT_TYPE } from "@/layered/service/log/layout";
import { Selectable } from "@/layered/service/log/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/layered/service/log/imex";
export * from "@/layered/service/log/layout";
export * from "@/layered/service/log/link";
export * from "@/layered/service/log/ontology";
export * from "@/layered/service/log/palette";
export * from "@/layered/service/log/Selectable";
export * from "@/layered/service/log/toolbar";
export * from "@/layered/service/log/useCreate";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Log };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
