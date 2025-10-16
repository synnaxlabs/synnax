// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import {
  Arc,
  type ContextMenu as PContextMenu,
  Form,
  type List,
} from "@synnaxlabs/pluto";

import { ContextMenu as CMenu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends PContextMenu.MenuProps {
  getItem: List.GetItem<string, arc.Arc>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const arcs = getItem(keys);
  const isEmpty = arcs.length === 0;
  const isSingle = arcs.length === 1;
  const ctx = Form.useContext();
  const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Arc",
    description: "Deleting this arc will permanently remove it.",
  });
  const { update: del } = Arc.useDelete();

  const handleRename = () => {
    rename({ initialValue: arcs[0].name }, { icon: "Arc", name: "Arc.Rename" })
      .then((renamed) => {
        if (renamed == null) return;
        ctx.set("name", renamed);
      })
      .catch(console.error);
  };
  const handleDelete = () => {
    confirm(arcs)
      .then((confirmed) => {
        if (confirmed) del(arcs.map((a) => a.key));
      })
      .catch(console.error);
  };

  return (
    <>
      {isSingle && <CMenu.RenameItem onClick={handleRename} />}
      {!isEmpty && <CMenu.DeleteItem onClick={handleDelete} />}
    </>
  );
};
