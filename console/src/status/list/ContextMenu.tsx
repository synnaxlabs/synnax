// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, Icon, Menu as PMenu, Status } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { useSelectFavoriteSet } from "@/status/selectors";
import { addFavorites, removeFavorites } from "@/status/slice";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {}

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
  const handleSelect: PMenu.MenuProps["onChange"] = {
    delete: () => {
      handleError(async () => {
        const confirmed = await confirm(statuses);
        if (confirmed) del(keys);
      }, "Failed to delete status");
    },
    rename: () => {
      rename.update(statuses[0]);
    },
    favorite: () => {
      dispatch(addFavorites(keys));
    },
    unfavorite: () => {
      dispatch(removeFavorites(keys));
    },
  };

  const isEmpty = statuses.length === 0;
  const isSingle = statuses.length === 1;

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {anyNotFavorited && (
        <PMenu.Item itemKey="favorite">
          <Icon.StarFilled />
          Favorite
        </PMenu.Item>
      )}
      {anyFavorited && (
        <PMenu.Item itemKey="unfavorite">
          <Icon.StarOutlined />
          Unfavorite
        </PMenu.Item>
      )}
      {(anyFavorited || anyNotFavorited) && <PMenu.Divider />}
      {!isEmpty && <Menu.DeleteItem />}
      {isSingle && <Menu.RenameItem />}
    </PMenu.Menu>
  );
};
