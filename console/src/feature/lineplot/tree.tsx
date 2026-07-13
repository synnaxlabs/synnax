// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, ontology } from "@synnaxlabs/client";
import {
  Access,
  Icon,
  LinePlot as Base,
  Menu,
  Mosaic,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback } from "react";

import { useExport } from "@/feature/lineplot/export";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Layout } from "@/platform/layout";
import { create } from "@/platform/lineplot/layout";
import { Link } from "@/platform/link";
import { Node } from "@/platform/node";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useDelete = Tree.createUseDelete({
  type: "Line Plot",
  icon: "LinePlot",
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Session.LinePlot.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = Tree.createUseRename({
  query: Base.useRename,
  ontologyID: lineplot.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(Session.Layout.rename({ key, name }));
    rollbacks.push(() => store.dispatch(Session.Layout.rename({ key, name: oldName })));
    return { ...data, name };
  },
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const handleLink = Node.useCopyLinkToClipboard();
  const handleExport = useExport();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const firstID = ids[0];
  const isSingle = ids.length === 1;
  const first = getResource(firstID);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {isSingle && (
            <>
              <ContextMenu.RenameItem onClick={rename} />
              <Menu.Divider />
            </>
          )}
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
        </>
      )}
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {(hasUpdatePermission || hasDeletePermission) && <Menu.Divider />}
      {isSingle && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(first.id.key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
          />
          <Tree.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      handleError(async () => {
        const linePlot = await client.lineplots.retrieve({ key: resource.id.key });
        placeLayout(create({ key: linePlot.key, name: linePlot.name }));
      }, `Failed to select ${resource.name}`);
    },
    [client, placeLayout, handleError],
  );
};

const TreeItem = Tree.createItem({
  type: "lineplot",
  icon: <Icon.LinePlot />,
  hasChildren: false,
  useOnSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { lineplot: TreeItem } satisfies Tree.Items;
