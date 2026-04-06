// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Access, Divider, Icon, List, Menu, Ranger, Status } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { ContextMenu as CMenu } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { CreateChildRangeIcon } from "@/range/ContextMenu";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { useSelectKeys } from "@/range/selectors";
import { add, remove } from "@/range/slice";
import { fromClientRange } from "@/range/translate";

export const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
  const { getItem } = List.useUtilContext<ranger.Key, ranger.Range>();
  const ranges = getItem?.(keys) ?? [];
  const isNotEmpty = ranges.length !== 0;
  const isSingle = ranges.length === 1;
  const ids = ranger.ontologyID(keys);
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const placeLayout = Layout.usePlacer();
  const favoriteKeys = useSelectKeys();
  const someAreFavorites = ranges.some((r) => favoriteKeys.includes(r.key));
  const someAreNotFavorites = ranges.some((r) => !favoriteKeys.includes(r.key));
  const dispatch = useDispatch();
  const renameModal = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  const { update: del } = Ranger.useDelete();
  const { update: renameRange } = Ranger.useRename();
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: ranges[0].key }));
  };
  const handleFavorite = () => {
    dispatch(add({ ranges: fromClientRange(ranges) }));
  };
  const handleUnfavorite = () => {
    dispatch(remove({ keys: ranges.map((r) => r.key) }));
  };
  const handleError = Status.useErrorHandler();
  const handleLink = Cluster.useCopyLinkToClipboard();

  const handleDetails = () => {
    placeLayout({ ...OVERVIEW_LAYOUT, name: ranges[0].name, key: ranges[0].key });
  };
  const handleRename = () => {
    handleError(async () => {
      const renamed = await renameModal(
        { initialValue: ranges[0].name },
        { icon: "Range", name: "Range.Rename" },
      );
      if (renamed == null) return;
      renameRange({ key: ranges[0].key, name: renamed });
    }, "Failed to rename range");
  };
  const handleDelete = () => {
    handleError(async () => {
      const confirmed = await confirm(ranges);
      if (!confirmed) return;
      const keys = ranges.map((r) => r.key);
      dispatch(remove({ keys }));
      dispatch(Layout.remove({ keys }));
      del(keys);
    }, "Failed to delete range");
  };

  return (
    <CMenu.Menu>
      {isSingle && (
        <>
          <Menu.Item itemKey="details" onClick={handleDetails}>
            <Icon.Details />
            View details
          </Menu.Item>
          {hasUpdatePermission && <CMenu.RenameItem onClick={handleRename} />}
          {hasCreatePermission && (
            <Menu.Item itemKey="addChildRange" onClick={handleAddChildRange}>
              <CreateChildRangeIcon key="plot" />
              Create child range
            </Menu.Item>
          )}
          <Divider.Divider x />
        </>
      )}
      <CMenu.FavoriteItems
        anyFavorited={someAreFavorites}
        anyNotFavorited={someAreNotFavorites}
        onFavorite={handleFavorite}
        onUnfavorite={handleUnfavorite}
      />
      {(someAreFavorites || someAreNotFavorites) && <Divider.Divider x />}
      {hasDeletePermission && isNotEmpty && (
        <>
          <CMenu.DeleteItem onClick={handleDelete} />
          <Divider.Divider x />
        </>
      )}
      {isSingle && (
        <>
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({
                name: ranges[0].name,
                ontologyID: ranger.ontologyID(ranges[0].key),
              })
            }
          />
          <Divider.Divider x />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};
