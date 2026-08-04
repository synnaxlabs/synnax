// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NotFoundError,
  type ontology,
  type project,
  type Synnax,
} from "@synnaxlabs/client";
import { context, type Icon, Panel, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

export interface TabName extends FC<record.Empty> {}
export interface TabIcon extends Icon.FC {}
export interface Toolbar extends FC<record.Empty> {}
export interface Content extends FC<record.Empty> {}

export interface RestoreParams {
  client: Synnax;
  project: project.Key;
  resource: ontology.ID;
}

export interface Tab {
  Content: Content;
  Name: TabName;
  /** Represents the tab as a glyph alone, e.g. on the bottom toolbar button.
   * Rendered inside the tab's panel and tab scope. */
  Icon: TabIcon;
  Toolbar?: Toolbar;
  /**
   * Re-creates the tab's deleted resource from the corpse held by the client's
   * cache. Corpses keep their original keys, so restoring re-registers the
   * document under the same key and every reference to it works again. Absent for
   * types that cannot be restored; their tombstones offer Close only.
   */
  restore?: (params: RestoreParams) => Promise<void>;
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

// The selector chip is the canonical rename target: only it carries the tab-name
// DOM id. Secondary Name mounts (e.g. the toolbar header) set this false so the
// id stays unique and Text.edit keeps resolving to the chip.
const [NameEditTargetContext, useIsNameEditTarget] = context.create<boolean>({
  defaultValue: true,
  displayName: "Tabs.NameEditTargetContext",
});

export { NameEditTargetContext, useIsNameEditTarget };

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
    const isEditTarget = useIsNameEditTarget();
    const { key } = Panel.useSelectTabResource();
    service.useEnsureRetrieved({ key });
    const name = service.useSelectName({ key });
    const { update } = service.useRename();
    return (
      <>
        {icon}
        <Text.Editable
          id={isEditTarget ? tabNameID(tabKey) : undefined}
          value={name}
          onChange={(name) => update({ key, name })}
        />
      </>
    );
  };
  return Name;
};
