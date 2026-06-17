// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import {
  Access,
  CSS as PCSS,
  type Flux,
  type List,
  Menu,
  Panel,
  Panel as PlutoPanel,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu } from "@/components";
import { useConfirmDelete } from "@/ontology/hooks";
import { Session } from "@/panel/session";

interface PanelContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<panel.Key, panel.Panel>;
}

const PanelContextMenu = ({
  keys,
  getItem,
}: PanelContextMenuProps): ReactElement | null => {
  const ids = panel.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const confirm = useConfirmDelete({ type: "Panel" });
  const { update: del } = PlutoPanel.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<panel.Key | panel.Key[]>) => {
        const panelKeys = array.toArray(data);
        if (panelKeys.length === 0) return false;
        const panels = getItem(panelKeys);
        if (!(await confirm(panels))) return false;
        return data;
      },
      [getItem, confirm],
    ),
  });
  if (keys.length === 0) return null;
  const [key] = keys;
  return (
    <CMenu.Menu>
      {hasUpdatePermission && keys.length === 1 && (
        <>
          <CMenu.RenameItem onClick={() => Text.edit(PCSS.B(`tab-${key}`))} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && (
        <>
          <CMenu.DeleteItem onClick={() => del(keys)} />
          <Menu.Divider />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};

// PanelTabName renders a panel tab's name from the live Flux record. Renames
// leave the panel key unchanged, so the memoized tab list does not recompute;
// subscribing per-tab here keeps the displayed name in sync (and matches the
// synchronized names used by the inner panel mosaic tabs).
const PanelTabName = ({ tabKey, name, ...rest }: Tabs.NameProps): ReactElement => {
  const { data } = PlutoPanel.useRetrieve({ key: tabKey });
  return <Tabs.DefaultName tabKey={tabKey} name={data?.name ?? name} {...rest} />;
};

const renderPanelTabName: Tabs.NameRenderProp = (props) => <PanelTabName {...props} />;

export const Selector = (): ReactElement | null => {
  const dispatch = useDispatch();
  const selected = Session.useSelectSelected();
  const { data, retrieve, getItem } = PlutoPanel.useList();
  useEffect(() => retrieve({}), [retrieve]);

  const tabs = useMemo<Tabs.Tab[]>(() => {
    const out: Tabs.Tab[] = [];
    for (const key of data) {
      const p = getItem(key);
      if (p == null || Array.isArray(p)) continue;
      out.push({ tabKey: p.key, name: p.name, closable: true, editable: true });
    }
    return out;
  }, [data, getItem]);

  const handleSelect = useCallback(
    (key: string) => {
      dispatch(Session.select({ key }));
    },
    [dispatch],
  );

  const { update: create } = Panel.useCreate();
  const handleCreate = useCallback(() => {
    create({ name: "New Panel" });
  }, []);

  const { update: rename } = PlutoPanel.useRename();
  const handleRename = useCallback(
    (key: string, name: string) => rename({ key, name }),
    [rename],
  );

  useEffect(() => {
    if (tabs.length === 0) return;
    if (selected != null && tabs.some((t) => t.tabKey === selected)) return;
    dispatch(Session.select({ key: tabs[0].tabKey }));
  }, [selected, tabs, dispatch]);

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <PanelContextMenu {...props} getItem={getItem} />,
    [getItem],
  );

  const providerValue = useMemo<Tabs.ContextValue>(
    () => ({
      tabs,
      selected: selected ?? undefined,
      closable: true,
      onSelect: handleSelect,
      onCreate: handleCreate,
      onRename: handleRename,
      tabName: renderPanelTabName,
    }),
    [tabs, selected, handleSelect, handleCreate, handleRename],
  );
  return (
    <Tabs.Provider value={providerValue}>
      <Tabs.Selector size="medium" variant="pill" contextMenu={contextMenu} />
    </Tabs.Provider>
  );
};
