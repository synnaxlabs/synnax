// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, useContext } from "react";

import { type ContextMenuRenderer, type Renderer } from "@/layout/slice";

export type Renderers = Record<string, Renderer>;

const RendererContext = createContext<Renderers>({});

export const RendererProvider = RendererContext.Provider;

export const useLayoutRenderer = (type: string): Renderer => {
  const r = useContext(RendererContext)[type];
  if (r == null) throw new Error(`no renderer for layout type ${type}`);
  return r;
};

export const useOptionalRenderer = (type: string): Renderer | null =>
  useContext(RendererContext)[type] ?? null;

export type ContextMenus = Record<string, ContextMenuRenderer>;

const ContextMenuContext = createContext<ContextMenus>({});

export const ContextMenuProvider = ContextMenuContext.Provider;

export const useContextMenuRenderer = (type: string): ContextMenuRenderer | null =>
  useContext(ContextMenuContext)[type] ?? null;
