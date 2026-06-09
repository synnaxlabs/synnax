// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { createContext, use } from "react";

// RendererView is a view-tab renderer's connection to the tab that hosts it. A view
// tab (panel.TabView) has no backing core resource, so its display name and opaque
// args live on the tab itself; this exposes them to the renderer and lets it write
// changes back through the panel document (update). It is null whenever a renderer is
// mounted outside a panel view tab — e.g. a standalone window or a mosaic layout — in
// which case the renderer sources its state from the layout slice as before.
export interface RendererView {
  // key is the hosting tab's key (equal to the renderer's layoutKey for view tabs).
  key: string;
  // name is the view's current display name.
  name: string;
  // args is the view's opaque, Console-owned payload.
  args: record.Unknown;
  // update merges a patch into the view and persists it through the panel document.
  update: (patch: { name?: string; args?: record.Unknown }) => void;
}

const Context = createContext<RendererView | null>(null);
Context.displayName = "Layout.RendererView";

export const RendererViewProvider = Context.Provider;

// useRendererView returns the current view tab host, or null when the renderer is not
// mounted inside a panel view tab.
export const useRendererView = (): RendererView | null => use(Context);
