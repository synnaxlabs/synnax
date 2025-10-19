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
  type ContextMenu as PContextMenu,
  type Flux,
  type List,
  Status,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ContextMenu as CMenu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends PContextMenu.MenuProps {
  getItem: List.GetItem<string, status.Status>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const statuses = getItem(keys);
  const isEmpty = statuses.length === 0;
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
  const handleDelete = () => {
    handleError(async () => {
      const confirmed = await confirm(statuses);
      if (confirmed) del(statuses.map((s) => s.key));
    }, "Failed to delete status");
  };
  const handleRename = useCallback(() => {
    rename.update(statuses[0]);
  }, [rename, statuses]);

  return (
    <>
      {!isEmpty && <CMenu.DeleteItem onClick={handleDelete} showBottomDivider />}
      <CMenu.RenameItem onClick={handleRename} />
    </>
  );
};
