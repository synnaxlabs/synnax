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

export interface TabName extends FC<Omit<Panel.MosaicTabNameProps, "name">> {}

export interface Toolbar extends FC {}

export interface ContentProps {
  visible: boolean;
}

export interface Content extends FC<ContentProps> {}

export interface Tab {
  Content: Content;
  Name: TabName;
  Toolbar?: Toolbar;
}

export interface Tabs extends Record<string, Tab> {}

const [RendererContext, useRendererContext] = context.create<Tabs>({
  defaultValue: {},
  displayName: "Tabs.RendererContext",
});

export { RendererContext };

// useTab resolves the registered Tab for the active tab's type, sourced from the
// surrounding panel scope. Throws when no renderer is registered for the type.
export const useTab = (): Tab => {
  const type = Panel.useSelectTabType({});
  const renderer = useRendererContext()[type];
  if (renderer == null) throw new NotFoundError(`no renderer for tab type ${type}`);
  return renderer;
};

export interface CreateStaticTabNameParams {
  name: string;
  icon: Icon.ReactElement;
}

export const createStaticTabName = (params: CreateStaticTabNameParams): TabName => {
  const Name: TabName = (props) => <Panel.DefaultTabName {...props} {...params} />;
  return Name;
};
