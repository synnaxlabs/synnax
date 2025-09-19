// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { type List, Menu as PMenu, Status } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
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

  const handleSelect: PMenu.MenuProps["onChange"] = {
    delete: () => {
      handleError(async () => {
        const confirmed = await confirm(statuses);
        if (confirmed) del(statuses.map((s) => s.key));
      }, "Failed to delete status");
    },
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {!isEmpty && <Menu.DeleteItem />}
    </PMenu.Menu>
  );
};