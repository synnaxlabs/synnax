// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import {
  type Flux,
  Icon,
  type List,
  Menu as PMenu,
  Status,
} from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";

import { Menu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { selectIsFavorite } from "@/status/selectors";
import { toggleFavorite } from "@/status/slice";
import { type RootState } from "@/store";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
  getItem: List.GetItem<string, status.Status>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const statuses = getItem(keys);
  const isEmpty = statuses.length === 0;
  const dispatch = useDispatch();

  // Check favorite status for all selected items
  const favoriteStates = useSelector((state: RootState) =>
    statuses.map((s) => selectIsFavorite(state, s.key)),
  );

  // If any are not favorited, show "Favorite". If any are favorited, show "Unfavorite"
  const anyFavorited = useMemo(() => favoriteStates.some((f) => f), [favoriteStates]);
  const anyNotFavorited = useMemo(
    () => favoriteStates.some((f) => !f),
    [favoriteStates],
  );

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
      [getItem],
    ),
  });
  const handleSelect: PMenu.MenuProps["onChange"] = {
    delete: () => {
      handleError(async () => {
        const confirmed = await confirm(statuses);
        if (confirmed) del(statuses.map((s) => s.key));
      }, "Failed to delete status");
    },
    rename: useCallback(() => {
      rename.update(statuses[0]);
    }, [rename, statuses]),
    toggleFavorite: useCallback(() => {
      statuses.forEach((s) => dispatch(toggleFavorite({ key: s.key })));
    }, [dispatch, statuses]),
  };

  // Determine which action to show based on favorite states
  const showFavorite = anyNotFavorited;
  const showUnfavorite = anyFavorited;
  const isSingle = statuses.length === 1;

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {!isEmpty && (showFavorite || showUnfavorite) && (
        <>
          <PMenu.Item itemKey="toggleFavorite">
            {showUnfavorite ? <Icon.StarFilled /> : <Icon.StarOutlined />}
            {showUnfavorite ? "Unfavorite" : "Favorite"}
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      {!isEmpty && <Menu.DeleteItem />}
      {isSingle && <Menu.RenameItem />}
    </PMenu.Menu>
  );
};
