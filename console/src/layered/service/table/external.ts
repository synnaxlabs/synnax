// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Table } from "@/layered/service/table/body/Table";
import { extract } from "@/layered/service/table/imex/export";
import { LAYOUT_TYPE } from "@/layered/service/table/layout";
import { Selectable } from "@/layered/service/table/Selectable";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/layered/service/table/imex";
export * from "@/layered/service/table/layout";
export * from "@/layered/service/table/link";
export * from "@/layered/service/table/ontology";
export * from "@/layered/service/table/palette";
export * from "@/layered/service/table/toolbar";
export * from "@/layered/service/table/useCreate";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Table };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
