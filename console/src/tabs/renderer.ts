// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type ComponentType, type FC } from "react";

export interface Renderer extends FC {}

export interface Renderers extends Record<string, Renderer> {}

const [RendererContext, useRendererContext] = context.create<Renderers>({
  defaultValue: {},
  displayName: "Tabs.RendererContext",
});

/** Provides the type-to-component registry consumed by {@link useRenderer}. */
export const RendererProvider = RendererContext;

/** Resolves the registered {@link Renderer} for the given tab type. */
export const useRenderer = (type: string): Renderer => {
  const r = useRendererContext()[type];
  if (r == null) throw new Error(`no renderer for tab type ${type}`);
  return r;
};

export interface ContextMenuProps {
  layoutKey: string;
}

/** A component that renders a custom context menu for a tab of a given type. */
export interface ContextMenuRenderer extends ComponentType<ContextMenuProps> {}

export interface ContextMenus extends Record<string, ContextMenuRenderer> {}

const [ContextMenuContext, useContextMenuContext] = context.create<ContextMenus>({
  defaultValue: {},
  displayName: "Tabs.ContextMenuContext",
});

export const ContextMenuProvider = ContextMenuContext;

export const useContextMenuRenderer = (type: string): ContextMenuRenderer | null =>
  useContextMenuContext()[type] ?? null;
