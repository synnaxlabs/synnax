// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";
import { Access, type Flux, Menu, Text, Workspace } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { ContextMenu as CMenu } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { useConfirmDelete } from "@/ontology/hooks";
import { type RootState } from "@/store";
import { useExport } from "@/workspace/export";
import { selectActiveKey } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

type GetItem = (key: workspace.Key) => workspace.Workspace | undefined;

const useDelete = (getItem: GetItem) => {
  const store = useStore<RootState>();
  const confirm = useConfirmDelete({ type: "Workspace" });
  return Workspace.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<Workspace.DeleteParams>) => {
        const keys = array.toArray(data);
        const workspaces = keys.map(getItem).filter((ws) => ws != null);
        return await confirm(workspaces);
      },
      [confirm, getItem],
    ),
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<Workspace.DeleteParams>) => {
        const keys = array.toArray(data);
        const activeKey = selectActiveKey(store.getState());
        if (activeKey == null || !keys.includes(activeKey)) return;
        store.dispatch(setActive(null));
        store.dispatch(Layout.clearWorkspace());
      },
      [store],
    ),
  });
};

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: GetItem;
}

export const ContextMenu = ({
  keys: [key],
  getItem,
}: ContextMenuProps): ReactElement => {
  const ws = getItem(key);
  const id = workspace.ontologyID(key);
  const hasUpdatePermission = Access.useUpdateGranted(id);
  const hasDeletePermission = Access.useDeleteGranted(id);
  const del = useDelete(getItem);
  const handleExport = useExport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  return (
    <CMenu.Menu>
      {ws != null && (
        <>
          {hasUpdatePermission && (
            <CMenu.RenameItem onClick={() => Text.edit(`text-${ws.key}`)} />
          )}
          {hasDeletePermission && (
            <CMenu.DeleteItem onClick={() => del.update(ws.key)} />
          )}
          <Menu.Divider />
          <Export.ContextMenuItem onClick={() => handleExport(ws.key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: ws.name, ontologyID: id })}
          />
          <Menu.Divider />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};
