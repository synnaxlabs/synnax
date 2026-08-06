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
import { context, type Flux, type Icon, Panel, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

export interface TabName extends FC<record.Empty> {}
export interface TabIcon extends Icon.FC {}
export interface Toolbar extends FC<record.Empty> {}
export interface Content extends FC<record.Empty> {}

export interface RestoreParams {
  client: Synnax;
  project: project.Key;
  resource: ontology.ID;
}

/** Reads the corpse of the surrounding tab's resource, null while it is present. */
export interface UseTombstone {
  (): Flux.Tombstone | null;
}

export interface Tab {
  Content: Content;
  Name: TabName;
  /** Represents the tab as a glyph alone. Rendered inside the tab's panel and
   * tab scope. */
  Icon: TabIcon;
  Toolbar?: Toolbar;
  /**
   * Re-creates the tab's deleted resource from the corpse held by the client's
   * cache. Corpses keep their original keys, so restoring re-registers the
   * document under the same key and every reference to it works again. Absent for
   * types that cannot be restored; their tombstones offer Close only.
   */
  restore?: (params: RestoreParams) => Promise<void>;
  /**
   * Reads whether the tab's resource has been deleted. Absent for view tabs,
   * which address no resource of their own.
   */
  useTombstone?: UseTombstone;
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

export interface TombstoneService {
  useTombstone: (query: { key: string }) => Flux.Tombstone | null;
}

/** Binds a domain's tombstone read to the surrounding tab's resource. */
export const createTombstoneReader = (service: TombstoneService): UseTombstone => {
  const useRead: UseTombstone = () => {
    const { key } = Panel.useSelectTabResource();
    return service.useTombstone({ key });
  };
  return useRead;
};

export interface ResourceGuardProps {
  /** Rendered in place of children while the tab's resource is deleted. */
  FallbackComponent: FC<Flux.Tombstone>;
  children: ReactElement;
}

interface DeletableProps extends ResourceGuardProps {
  useTombstone: UseTombstone;
}

// Split from ResourceGuard so the tombstone read is unconditional: a tab type
// addressing no resource must not change the hook count of the component that
// renders one that does.
const Deletable = ({
  useTombstone,
  FallbackComponent,
  children,
}: DeletableProps): ReactElement => {
  const tombstone = useTombstone();
  if (tombstone == null) return children;
  return <FallbackComponent {...tombstone} />;
};

/**
 * Renders children while the tab's resource exists and the fallback while it has
 * been deleted, swapping back on its own once the resource returns. A tab type
 * addressing no resource of its own always renders children.
 */
export const ResourceGuard = ({
  FallbackComponent,
  children,
}: ResourceGuardProps): ReactElement => {
  const { useTombstone } = useTab();
  if (useTombstone == null) return children;
  return (
    <Deletable useTombstone={useTombstone} FallbackComponent={FallbackComponent}>
      {children}
    </Deletable>
  );
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
