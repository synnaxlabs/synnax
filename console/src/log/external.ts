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
import { Content } from "@/log/content/Content";
import { extract } from "@/log/export";
import { LAYOUT_TYPE } from "@/log/layout";
import { Selectable } from "@/log/Selectable";
import { type Selector } from "@/selector";

export * from "@/log/content/Content";
export * from "@/log/export";
export * from "@/log/layout";
export * from "@/log/Selectable";
export * from "@/log/session/slice";
export * from "@/log/toolbar/Toolbar";
export * from "@/log/useCreate";
export * from "@/log/useName";

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };

export const LAYOUTS: Record<string, Layout.Renderer> = { [LAYOUT_TYPE]: Content };

export const SELECTABLES: Selector.Selectable[] = [Selectable];
