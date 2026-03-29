// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Arc, Icon, type List, Menu, Status } from "@synnaxlabs/pluto";

import { Editor } from "@/arc/editor";
import { translateGraphToConsole } from "@/arc/types/translate";
import { Cluster } from "@/cluster";
import { ContextMenu as CMenu } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<arc.Key, arc.Arc>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const ids = arc.ontologyID(keys);
  const canDelete = Access.useDeleteGranted(ids);
  const canEdit = Access.useUpdateGranted(ids);
  const someSelected = keys.length > 0;
  const isSingle = keys.length === 1;

  const placeLayout = Layout.usePlacer();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const renameModal = Modals.useRename();
  const { update: renameArc } = Arc.useRename();
  const confirm = useConfirmDelete({
    type: "Arc",
    description: "Deleting this arc will permanently remove it.",
  });
  const { update: del } = Arc.useDelete();

  const handleEdit = () => {
    const retrieved = getItem(keys[0]);
    if (retrieved == null)
      return addStatus({
        variant: "error",
        message: "Failed to open Arc editor",
        description: `Arc with key ${keys[0]} not found`,
      });
    const { name, key, text, mode } = retrieved;
    const graph = translateGraphToConsole(retrieved.graph);
    placeLayout(Editor.create({ key, name, graph, text, mode }));
  };

  const handleRename = () => {
    handleError(async () => {
      const a = getItem(keys[0]);
      if (a == null) return;
      const renamed = await renameModal(
        { initialValue: a.name },
        { icon: "Arc", name: "Arc.Rename" },
      );
      if (renamed == null) return;
      renameArc({ key: keys[0], name: renamed });
    }, "Failed to rename arc");
  };

  const handleDelete = () => {
    handleError(async () => {
      const arcs = getItem(keys);
      const confirmed = await confirm(arcs);
      if (confirmed) del(keys);
    }, "Failed to delete arc");
  };

  const handleCopyLink = () => {
    const name = getItem(keys[0])?.name;
    if (name == null) return;
    handleLink({ name, ontologyID: arc.ontologyID(keys[0]) });
  };

  return (
    <CMenu.Menu>
      {canEdit && isSingle && (
        <>
          <Menu.Item itemKey="edit" onClick={handleEdit}>
            <Icon.Edit />
            Edit
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {canEdit && isSingle && (
        <>
          <CMenu.RenameItem onClick={handleRename} />
          <Menu.Divider />
        </>
      )}
      {canDelete && someSelected && (
        <>
          <CMenu.DeleteItem onClick={handleDelete} />
          <Menu.Divider />
        </>
      )}
      {isSingle && (
        <>
          <Link.CopyContextMenuItem onClick={handleCopyLink} />
          <Menu.Divider />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};
