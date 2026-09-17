// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, ranger } from "@synnaxlabs/client";
import {
  Access,
  type Haul,
  Icon,
  Menu,
  Ranger,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { CreateChildRangeIcon } from "@/feature/range/ContextMenu";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const openTab = Panel.useOpenTab();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      handleError(async () => {
        const ranges = await client.ranges.retrieve([resource.id.key]);
        store.dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
        openTab({ variant: "resource", resource: resource.id });
      }, `Failed to select ${resource.name}`);
    },
    [client, store, openTab, handleError],
  );
};

const haulItems = (resource: ontology.Resource): Haul.Item[] => {
  const payload = resource.data as ranger.Payload | null | undefined;
  if (payload == null) return [];
  return [Ranger.createHaulItem(payload)];
};

const useRename = Tree.createUseRename({
  query: Ranger.useRename,
  ontologyID: ranger.ontologyID,
  convertKey: String,
});

const useDelete = Tree.createUseDelete({
  type: "Range",
  description: "Deleting a range also deletes its child ranges.",
  query: Session.Range.useDelete,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    client,
    store,
    handleError,
    openTab,
    selection: { ids },
    state: { getResource },
  } = props;
  const keys = ids.map((id) => id.key);
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const favoriteKeys = Session.Range.useSelectKeys();
  const someAreFavorites = keys.some((k) => favoriteKeys.includes(k));
  const someAreNotFavorites = keys.some((k) => !favoriteKeys.includes(k));
  const openCreate = Range.useCreateModal();
  const rename = useRename(props);
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const firstID = ids[0];
  const first = getResource(firstID);
  const singleResource = ids.length === 1;
  const handleFavorite = () => {
    handleError(async () => {
      const ranges = await client.ranges.retrieve(keys);
      store.dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
    }, "Failed to favorite ranges");
  };
  const handleUnfavorite = () => {
    store.dispatch(Session.Range.remove({ keys }));
  };
  return (
    <ContextMenu.Menu>
      {singleResource && (
        <Menu.Item
          itemKey="details"
          onClick={() => openTab({ variant: "resource", resource: firstID })}
        >
          <Icon.Details />
          View details
        </Menu.Item>
      )}
      <Menu.Divider />
      {singleResource && (
        <>
          {hasUpdatePermission && <ContextMenu.RenameItem onClick={rename} />}
          {hasCreatePermission && (
            <Menu.Item
              itemKey="addChildRange"
              onClick={() => openCreate({ parent: firstID.key })}
            >
              <CreateChildRangeIcon key="plot" />
              Create child range
            </Menu.Item>
          )}
        </>
      )}
      <Menu.Divider />
      <ContextMenu.FavoriteItems
        anyFavorited={someAreFavorites}
        anyNotFavorited={someAreNotFavorites}
        onFavorite={handleFavorite}
        onUnfavorite={handleUnfavorite}
      />
      <Menu.Divider />
      {singleResource && (
        <Link.CopyContextMenuItem
          onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "range",
  icon: <Icon.Range />,
  useOnSelect,
  canDrop: () => true,
  haulItems,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { range: TreeItem } satisfies Tree.Items;
