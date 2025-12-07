// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Arc, Form, Icon, type List, Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
  getItem: List.GetItem<string, arc.Arc>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const arcs = getItem(keys);
  const isEmpty = arcs.length === 0;
  const isSingle = arcs.length === 1;
  const ids = arc.ontologyID(keys);
  const canDeleteAccess = Access.useDeleteGranted(ids);
  const canEditAccess = Access.useUpdateGranted(ids);
  const ctx = Form.useContext();
  const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Arc",
    description: "Deleting this arc will permanently remove it.",
  });
  const { update: del } = Arc.useDelete();

  const handleSelect: PMenu.MenuProps["onChange"] = {
    rename: () => {
      rename({ initialValue: arcs[0].name }, { icon: "Arc", name: "Arc.Rename" })
        .then((renamed) => {
          if (renamed == null) return;
          ctx.set("name", renamed);
        })
        .catch(console.error);
    },
    delete: () => {
      confirm(arcs)
        .then((confirmed) => {
          if (confirmed) del(arcs.map((a) => a.key));
        })
        .catch(console.error);
    },
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {canEditAccess && isSingle && <Menu.RenameItem />}
      {canDeleteAccess && !isEmpty && (
        <PMenu.Item itemKey="delete">
          <Icon.Delete />
          Delete
        </PMenu.Item>
      )}
    </PMenu.Menu>
  );
};
