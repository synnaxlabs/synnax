// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Component, type Flux, Menu, Status } from "@synnaxlabs/pluto";
import { status as xstatus } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu as CMenu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { useSelectFavoriteSet } from "@/status/selectors";
import { addFavorites, removeFavorites } from "@/status/slice";

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
  const q = Status.useRetrieveMultiple({ keys });
  const dispatch = useDispatch();
  const favoriteSet = useSelectFavoriteSet();
  const ids = status.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);

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
  const getCopyText = useCallback(() => {
    if (q.variant !== "success") return "";
    return q.data.map((s) => xstatus.toString(s)).join("\n\n");
  }, [q]);

  if (q.variant !== "success") return null;
  const statuses = q.data;
  const isEmpty = statuses.length === 0;
  const isSingle = statuses.length === 1;

  return (
    <CMenu.Menu>
      <CMenu.FavoriteItems
        anyFavorited={anyFavorited}
        anyNotFavorited={anyNotFavorited}
        onFavorite={() => dispatch(addFavorites(keys))}
        onUnfavorite={() => dispatch(removeFavorites(keys))}
      />
      {(anyFavorited || anyNotFavorited) && <Menu.Divider />}
      {!isEmpty && (
        <>
          <Menu.CopyItem
            itemKey="copyDiagnostics"
            text={getCopyText}
            successMessage="Copied diagnostics to clipboard"
          >
            Copy Diagnostics
          </Menu.CopyItem>
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && !isEmpty && (
        <CMenu.DeleteItem
          onClick={() => {
            handleError(async () => {
              const confirmed = await confirm(statuses);
              if (confirmed) del(keys);
            }, "Failed to delete status");
          }}
        />
      )}
      {hasUpdatePermission && isSingle && (
        <CMenu.RenameItem onClick={() => rename.update(statuses[0])} />
      )}
    </CMenu.Menu>
  );
};

export const contextMenu = Component.renderProp(ContextMenu);
