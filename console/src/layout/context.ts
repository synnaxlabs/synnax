// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, use } from "react";

import { type ContextMenuRenderer, type Renderer } from "@/layout/slice";

export interface Renderers extends Record<string, Renderer> {}

const RendererContext = createContext<Renderers>({});

export const RendererProvider = RendererContext;

export const useRenderer = (type: string): Renderer => {
  const r = use(RendererContext)[type];
  if (r == null) throw new Error(`no renderer for layout type ${type}`);
  return r;
};

export const useOptionalRenderer = (type: string): Renderer | null =>
  use(RendererContext)[type] ?? null;

export interface ContextMenus extends Record<string, ContextMenuRenderer> {}

const ContextMenuContext = createContext<ContextMenus>({});

export const ContextMenuProvider = ContextMenuContext;

export const useContextMenuRenderer = (type: string): ContextMenuRenderer | null =>
  use(ContextMenuContext)[type] ?? null;
