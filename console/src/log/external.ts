// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { type Layout } from "@/layout";
import { extract } from "@/log/export";
import { LAYOUT_TYPE, Log, Selectable } from "@/log/Log";
import { type Selector } from "@/selector";

export * from "@/log/export";
export * from "@/log/Log";
export * from "@/log/selectors";
export * from "@/log/slice";
export * from "@/log/Toolbar";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Log };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
