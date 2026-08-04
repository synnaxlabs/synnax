// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Component, type Flux, Icon, Menu, Status } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { ContextMenu as Base } from "@/platform/context-menu";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
  const q = Status.useRetrieveMultiple({ keys });
  const dispatch = Session.useDispatch();
  const favoriteSet = Session.Status.useSelectFavoriteSet();
  const ids = status.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);

  const confirm = Tree.useConfirmDelete({
    type: "Status",
    description: "This action cannot be undone.",
  });
  const { update: del } = Status.useDelete();
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();
  const rename = Status.useRename({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<Status.RenameParams>) => {
        const renamed = await renameModal({
          initialValue: data.name,
          title: "Status.Rename",
          icon: <Icon.Status />,
        });
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
    return q.data.map((s) => status.toString(s)).join("\n\n");
  }, [q]);

  if (q.variant !== "success") return null;
  const statuses = q.data;
  const isEmpty = statuses.length === 0;
  const isSingle = statuses.length === 1;

  return (
    <Base.Menu>
      {hasUpdatePermission && isSingle && (
        <Base.RenameItem onClick={() => rename.update(statuses[0])} />
      )}
      <Menu.Divider />
      <Base.FavoriteItems
        anyFavorited={anyFavorited}
        anyNotFavorited={anyNotFavorited}
        onFavorite={() => dispatch(Session.Status.addFavorites(keys))}
        onUnfavorite={() => dispatch(Session.Status.removeFavorites(keys))}
      />
      <Menu.Divider />
      {!isEmpty && (
        <Menu.CopyItem
          itemKey="copyDiagnostics"
          text={getCopyText}
          successMessage="Copied diagnostics to clipboard"
        >
          Copy diagnostics
        </Menu.CopyItem>
      )}
      <Menu.Divider />
      {hasDeletePermission && !isEmpty && (
        <Base.DeleteItem
          onClick={() => {
            handleError(async () => {
              const confirmed = await confirm(statuses);
              if (confirmed) del(keys);
            }, "Failed to delete status");
          }}
        />
      )}
      <Menu.Divider />
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};

export const contextMenu = Component.renderProp(ContextMenu);
