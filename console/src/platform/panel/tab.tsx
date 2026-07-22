// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { context, type Icon, Panel, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

export interface TabName extends FC<record.Empty> {}
export interface Toolbar extends FC<record.Empty> {}
export interface Content extends FC<record.Empty> {}

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

export interface UseTabReturn extends Tab {
  type: string;
}

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

export const createStaticTabName = ({
  icon,
  name,
}: CreateStaticTabNameParams): TabName => {
  const Name: TabName = () => (
    <>
      {icon}
      <Text.Text>{name}</Text.Text>
    </>
  );
  return Name;
};

/** tabNameID returns the DOM id of a tab's editable name, the target of Text.edit. */
export const tabNameID = (tabKey: string): string => `tab-name-${tabKey}`;

/**
 * editTabName starts an in-place edit of the tab's name. Tabs with static names carry no
 * id, so editing one is a no-op.
 */
export const editTabName = (tabKey: string): void => {
  const id = tabNameID(tabKey);
  if (document.getElementById(id) == null) return;
  Text.edit(id);
};

export interface EditableTabNameService {
  useEnsureRetrieved: (args: { key: string }) => void;
  useSelectName: (args: { key: string }) => string;
  useRename: () => { update: (args: { key: string; name: string }) => void };
}

export const createEditableTabName = (
  service: EditableTabNameService,
  icon: Icon.ReactElement,
): TabName => {
  const Name: TabName = () => {
    const tabKey = Panel.useTabKey();
    const { key } = Panel.useSelectTabResource();
    service.useEnsureRetrieved({ key });
    const name = service.useSelectName({ key });
    const { update } = service.useRename();
    return (
      <>
        {icon}
        <Text.Editable
          id={tabNameID(tabKey)}
          value={name}
          onChange={(name) => update({ key, name })}
        />
      </>
    );
  };
  return Name;
};
