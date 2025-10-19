// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  Flux,
  Group,
  Icon,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components/context-menu";
import { ContextMenuItem } from "@/group/ContextMenuItem";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseRename } from "@/ontology/createUseRename";

const useRename = createUseRename({
  query: Group.useRename,
  ontologyID: group.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids },
    state: { getResource, nodes },
  } = props;
  const ungroup = useUngroupSelection();
  const createEmptyGroup = useCreateEmpty(props);
  const copyLink = Cluster.useCopyLinkToClipboard();
  const firstID = ids[0];
  const firstResource = getResource(firstID);
  const rename = useRename(props);
  const isSingle = ids.length === 1;
  const isDelete = ids.every((id) => {
    const node = Tree.findNode({ tree: nodes, key: ontology.idToString(id) });
    return node?.children == null || node?.children.length === 0;
  });
  const handleUngroup = () => ungroup.update(props);
  const handleLink = () => copyLink({ name: firstResource.name, ontologyID: firstID });
  return (
    <>
      {isSingle && (
        <>
          <ContextMenu.RenameItem onClick={rename} showBottomDivider />
          <PContextMenu.Item onClick={createEmptyGroup}>
            <Icon.Group />
            New group
          </PContextMenu.Item>
        </>
      )}
      <ContextMenuItem {...props} />
      {isDelete ? (
        <ContextMenu.DeleteItem onClick={handleUngroup} showBottomDivider />
      ) : (
        <PContextMenu.Item onClick={handleUngroup} showBottomDivider>
          <Icon.Group />
          Ungroup
        </PContextMenu.Item>
      )}
      {isSingle && (
        <>
          <Ontology.CopyContextMenuItem {...props} />
          <Link.CopyContextMenuItem onClick={handleLink} showBottomDivider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </>
  );
};

interface UngroupParams extends Ontology.TreeContextMenuProps {
  prevNodes?: Tree.Node<string>[];
}

const baseUngroup = Flux.createUpdate<UngroupParams, Group.FluxSubStore>({
  name: Group.RESOURCE_NAME,
  verbs: {
    present: "ungroup",
    past: "ungrouped",
    participle: "ungrouping",
  },
  update: async ({ client, data: args }) => {
    const { selection, prevNodes } = args;
    if (selection.parentID == null || prevNodes == null) return args;
    const resourceIDStrings = new Set(
      selection.ids.map((id) => ontology.idToString(id)),
    );
    for (const id of selection.ids) {
      const children =
        Tree.findNode({ tree: prevNodes, key: ontology.idToString(id) })?.children ??
        [];
      const parentID = selection.parentID;
      const childKeys = ontology.parseIDs(
        children.map(({ key }) => key).filter((k) => !resourceIDStrings.has(k)),
      );
      await client.ontology.moveChildren(id, parentID, ...childKeys);
    }
    await client.ontology.groups.delete(selection.ids.map((id) => id.key));
    return args;
  },
});

const beforeUngroup = async ({ data }: Flux.BeforeUpdateParams<UngroupParams>) => {
  const {
    selection,
    state: { shape, nodes, setNodes },
  } = data;
  if (selection.parentID == null) return false;
  // Sort the groups by depth that way deeper nested groups are ungrouped first.
  selection.ids.sort(
    (a, b) =>
      Tree.getDepth(ontology.idToString(a), shape) -
      Tree.getDepth(ontology.idToString(b), shape),
  );
  const prevNodes = Tree.deepCopy(nodes);
  const nextNodes = [
    ...selection.ids.reduce(
      (acc, id) => {
        const key = ontology.idToString(id);
        const children = Tree.findNode({ tree: nodes, key })?.children ?? [];
        acc = Tree.moveNode({
          tree: acc,
          destination: ontology.idsEqual(selection.parentID, selection.rootID)
            ? null
            : ontology.idToString(selection.parentID),
          keys: children.map((c) => c.key),
        });
        acc = Tree.removeNode({ tree: acc, keys: key });
        return [...acc];
      },
      [...nodes],
    ),
  ];
  setNodes(nextNodes);
  return { ...data, prevNodes };
};

const afterUngroupFailure = async ({
  data: {
    prevNodes,
    state: { setNodes },
  },
}: Flux.AfterFailureParams<UngroupParams>) => {
  if (prevNodes != null) setNodes(prevNodes);
};

const useUngroupSelection = () =>
  baseUngroup.useUpdate({
    beforeUpdate: beforeUngroup,
    afterFailure: afterUngroupFailure,
  });

const useCreateEmpty = ({
  selection: {
    ids: [firstID],
  },
  state: { nodes: tree, setNodes, setResource, expand },
}: Ontology.TreeContextMenuProps) => {
  const { update } = Group.useCreate({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Group.CreateParams>) => {
        const newID = group.ontologyID(uuid.create());
        const newIDString = ontology.idToString(newID);
        const res: ontology.Resource = { key: newIDString, id: newID, name: "" };
        const node: Tree.Node<string> = { key: newIDString, children: [] };
        setResource(res);
        const destination = ontology.idToString(data.parent);
        expand(destination);
        setNodes([...Tree.setNode({ tree, destination, additions: node })]);
        rollbacks.push(() =>
          setNodes([...Tree.removeNode({ tree, keys: newIDString })]),
        );
        const [name, renamed] = await Text.asyncEdit(newIDString);
        if (!renamed || name === "") return false;
        return { ...data, key: newID.key, name };
      },
      [tree, setNodes, setResource, expand],
    ),
  });
  return useCallback(
    () => update({ key: uuid.create(), name: "", parent: firstID }),
    [firstID],
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "group",
  icon: <Icon.Group />,
  canDrop: () => true,
  // This haul item allows the group to be dragged between nodes in the tree.
  haulItems: ({ id }) => [id],
  TreeContextMenu,
};
