// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Component,
  ContextMenu as PContextMenu,
  type Flux,
  Icon,
  Status,
} from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { useSelectFavoriteSet } from "@/status/selectors";
import { addFavorites, removeFavorites } from "@/status/slice";

export interface ContextMenuProps extends PContextMenu.MenuProps {}

export const ContextMenu = ({ keys }: ContextMenuProps) => {
  const q = Status.useRetrieveMultiple({ keys });
  const dispatch = useDispatch();
  const favoriteSet = useSelectFavoriteSet();

  const confirm = useConfirmDelete({
    type: "Status",
    description: "This action cannot be undone.",
  });
  const { update: del } = Status.useDelete();
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();
  const rename = Status.useRename({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<Status.RenameParams>) => {
        const renamed = await renameModal(
          { initialValue: data.name },
          { icon: "Status", name: "Status.Rename" },
        );
        if (renamed == null) return false;
        return { ...data, name: renamed };
      },
      [renameModal],
    ),
  });

  const anyFavorited = useMemo(
    () => keys.some((k) => favoriteSet.has(k)),
    [favoriteSet, keys],
  );
  const anyNotFavorited = useMemo(
    () => keys.some((k) => !favoriteSet.has(k)),
    [favoriteSet, keys],
  );
  if (q.variant !== "success") return null;
  const statuses = q.data;
  const handleDelete = () => {
    handleError(async () => {
      const confirmed = await confirm(statuses);
      if (confirmed) del(keys);
    }, "Failed to delete status");
  };
  const handleFavorite = () => {
    dispatch(addFavorites(keys));
  };
  const handleUnfavorite = () => {
    dispatch(removeFavorites(keys));
  };
  const handleRename = useCallback(() => {
    rename.update(statuses[0]);
  }, [rename, statuses]);

  const isEmpty = statuses.length === 0;
  const isSingle = statuses.length === 1;

  return (
    <>
      {anyNotFavorited && (
        <PContextMenu.Item onClick={handleFavorite}>
          <Icon.StarFilled />
          Favorite
        </PContextMenu.Item>
      )}
      {anyFavorited && (
        <PContextMenu.Item onClick={handleUnfavorite}>
          <Icon.StarOutlined />
          Unfavorite
        </PContextMenu.Item>
      )}
      {(anyFavorited || anyNotFavorited) && <PContextMenu.Divider />}
      {!isEmpty && <CMenu.DeleteItem onClick={handleDelete} showBottomDivider />}
      {isSingle && <CMenu.RenameItem onClick={handleRename} showBottomDivider />}
      <CMenu.ReloadConsoleItem />
    </>
  );
};

export const contextMenuRenderProp = Component.renderProp(ContextMenu);
