// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { type Selector } from "@/selector";
import { ContextMenu } from "@/spectrogram/ContextMenu";
import { LAYOUT_TYPE } from "@/spectrogram/layout";
import { Selectable } from "@/spectrogram/Selectable";
import { Spectrogram } from "@/spectrogram/Spectrogram";

export * from "@/spectrogram/layout";
export * from "@/spectrogram/middleware";
export * from "@/spectrogram/selectors";
export * from "@/spectrogram/slice";
export * from "@/spectrogram/toolbar";

export const CONTEXT_MENUS: Record<string, Layout.ContextMenuRenderer> = {
  [LAYOUT_TYPE]: ContextMenu,
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Spectrogram,
};

export const SELECTABLES: Selector.Selectable[] = [Selectable];
