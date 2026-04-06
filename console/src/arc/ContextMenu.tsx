// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import {
  Access,
  Arc,
  type Flux,
  Icon,
  type List,
  Menu,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Editor } from "@/arc/editor";
import { translateGraphToConsole } from "@/arc/types/translate";
import { Cluster } from "@/cluster";
import { ContextMenu as CMenu } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<arc.Key, arc.Arc>;
  textIdPrefix?: string;
}

export const ContextMenu = ({
  keys,
  getItem,
  textIdPrefix = "text",
}: ContextMenuProps) => {
  const ids = arc.ontologyID(keys);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const someSelected = keys.length > 0;
  const isSingle = keys.length === 1;

  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  const addStatus = Status.useAdder();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const confirm = useConfirmDelete({
    type: "Arc",
    description: "Deleting this Arc will permanently remove it.",
  });
  const { update: del } = Arc.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<arc.Key | arc.Key[]>) => {
        const arcKeys = array.toArray(data);
        if (arcKeys.length === 0) return false;
        const arcs = getItem(arcKeys);
        if (!(await confirm(arcs))) return false;
        dispatch(Layout.remove({ keys: arcKeys }));
        return data;
      },
      [getItem, dispatch],
    ),
  });

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

  const handleCopyLink = () => {
    const name = getItem(keys[0])?.name;
    if (name == null) return;
    handleLink({ name, ontologyID: arc.ontologyID(keys[0]) });
  };

  return (
    <CMenu.Menu>
      {hasUpdatePermission && isSingle && (
        <>
          <Menu.Item itemKey="edit" onClick={handleEdit}>
            <Icon.Edit />
            Edit
          </Menu.Item>
          <Menu.Divider />
          <CMenu.RenameItem onClick={() => Text.edit(`${textIdPrefix}-${keys[0]}`)} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && someSelected && (
        <>
          <CMenu.DeleteItem onClick={() => del(keys)} />
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
