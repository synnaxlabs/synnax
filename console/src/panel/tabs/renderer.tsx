// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { context, type Icon, Panel } from "@synnaxlabs/pluto";
import { type FC } from "react";

export interface Name extends FC<Panel.MosaicTabNameRenderProps> {}

export interface Toolbar extends FC {}

export interface Content extends FC {}

export interface Renderer {
  Content: Content;
  Name: Name;
  Toolbar?: Toolbar;
}

export interface Renderers extends Record<string, Renderer> {}

const [RendererContext, useRendererContext] = context.create<Renderers>({
  defaultValue: {},
  displayName: "Tabs.RendererContext",
});

export { RendererContext };

export const useRenderer = (): Renderer => {
  const type = Panel.useSelectTabType({});
  const renderer = useRendererContext()[type];
  if (renderer == null) throw new NotFoundError(`no renderer for tab type ${type}`);
  return renderer;
};

export const useName = (): Name => {
  const type = Panel.useSelectTabType({});
  const renderer = useRendererContext()[type];
  if (renderer == null) throw new NotFoundError(`no renderer for tab type ${type}`);
  return renderer.Name;
};

interface StaticNameParams {
  name: string;
  icon: Icon.ReactElement;
}

export const createStaticName = (params: StaticNameParams): Name => {
  const N: Name = (props) => <Panel.DefaultTabName {...params} {...props} />;
  N.displayName = params.name;
  return N;
};
