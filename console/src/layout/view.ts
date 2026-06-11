// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type panel } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { createContext, use } from "react";

// ResolvedContent is content a renderer resolves into its hosting tab: an
// ontology-backed resource (for visualizations) or an inline view (for arg-driven
// views such as task forms). The hosting panel routes these to SetTabResource /
// SetTabView respectively.
export type ResolvedContent = { resource: ontology.ID } | { view: panel.TabView };

// RendererView is a renderer's connection to the host that mounts it: the hosting
// panel tab (name and opaque args live on the tab, written back through the panel
// document) or a window/modal layout (name and args live in the layout slice).
// Hosts always provide it, so arg-driven renderers read one channel regardless of
// where they are mounted.
export interface RendererView {
  // key is the hosting tab's key (equal to the renderer's layoutKey for view tabs).
  key: string;
  // name is the view's current display name.
  name: string;
  // args is the view's opaque, Console-owned payload.
  args: record.Unknown;
  // update merges a patch into the view and persists it through the panel document.
  update: (patch: { name?: string; args?: record.Unknown }) => void;
  // resolve replaces the hosting tab's content entirely — used by picker-style views
  // (e.g. the task-type selector) to swap themselves for the content the user chose,
  // keeping the tab's identity and position. Absent when the host has no notion of
  // swappable content (windows and modals), in which case pickers fall back to
  // placing layouts.
  resolve?: (content: ResolvedContent) => void;
}

const Context = createContext<RendererView | null>(null);
Context.displayName = "Layout.RendererView";

export const RendererViewProvider = Context.Provider;

// useRendererView returns the current view tab host, or null when the renderer is not
// mounted inside a panel view tab.
export const useRendererView = (): RendererView | null => use(Context);
