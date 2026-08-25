// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Component, List, Menu, Status, Text } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { ContextMenu as Base } from "@/platform/context-menu";
import { Errors } from "@/platform/errors";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  /** Builds the DOM id of a row's editable name. Lists rendering the same statuses
   * under different ids (e.g. the favorites toolbar) pass their own builder. */
  nameID?: (key: status.Key) => string;
}

const Internal = ({ keys, nameID = List.itemNameID }: ContextMenuProps) => {
  const statuses = Status.useMultiple({ keys });
  const dispatch = Session.useDispatch();
  const favoriteSet = Session.Status.useSelectFavoriteSet();
  const ids = status.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);

  const confirm = Modals.useConfirmDelete({ type: "Status" });
  const { update: del } = Status.useDelete();
  const handleError = Status.useErrorHandler();

  const anyFavorited = useMemo(
    () => keys.some((k) => favoriteSet.has(k)),
    [favoriteSet, keys],
  );
  const anyNotFavorited = useMemo(
    () => keys.some((k) => !favoriteSet.has(k)),
    [favoriteSet, keys],
  );
  const getCopyText = useCallback(
    () => statuses.map((s) => status.toString(s)).join("\n\n"),
    [statuses],
  );

  const isEmpty = statuses.length === 0;
  const isSingle = statuses.length === 1;

  return (
    <Base.Menu>
      {hasUpdatePermission && isSingle && (
        <Base.RenameItem onClick={() => Text.edit(nameID(statuses[0].key))} />
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

export const ContextMenu = (props: ContextMenuProps) => (
  <Errors.SuspenseBoundary loading={null}>
    <Internal {...props} />
  </Errors.SuspenseBoundary>
);

export const contextMenu = Component.renderProp(ContextMenu);
