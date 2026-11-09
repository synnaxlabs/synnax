// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, resource } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Flux, Group } from "@synnaxlabs/pluto";
import { uuid, verbs } from "@synnaxlabs/x";

import { moveRequests } from "@/platform/group/move";
import { placement } from "@/platform/group/placement";
import { type Tree } from "@/platform/tree";

interface CreateParams extends Tree.ContextMenuProps {
  group: Pick<group.Group, "key" | "name">;
}

const base = Flux.createUpdate<CreateParams>({
  name: Group.RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    const {
      selection: { parentID, rootID, top },
      group: { name, key },
    } = data;
    if (parentID == null)
      throw new Error("Cannot group items that sit under more than one parent");
    await client.batch.send([
      client.groups.createRequest({ key, name, ...placement(parentID, rootID) }),
      ...moveRequests(client, top, group.resourceID(key)),
    ]);
    return data;
  },
});

const beforeUpdate = async ({
  data,
  rollbacks,
}: Flux.BeforeUpdateParams<CreateParams>) => {
  const {
    selection: { top },
    state: { nodes, setNodes, setSelection, setResource },
    group: { key },
  } = data;
  const newID = group.resourceID(key);
  const newIDString = resource.idToString(newID);
  const prevNodes = PTree.deepCopy(nodes);
  rollbacks.push(() => setNodes(prevNodes));
  const res: Tree.Resource = { key: newIDString, id: newID, name: "" };
  setResource(res);
  // The new group shows where the selection shows. In flat search results, that is the
  // top of the tree, while the real parent is a group.
  const destination =
    PTree.findNodeParent({
      tree: nodes,
      key: resource.idToString(top[0]),
    })?.key ?? null;
  let nextNodes = PTree.setNode({
    tree: nodes,
    destination,
    additions: { key: newIDString, children: [] },
  });
  nextNodes = PTree.moveNode({
    tree: nextNodes,
    destination: newIDString,
    keys: top.map((id) => resource.idToString(id)),
  });
  setNodes([...nextNodes]);
  setSelection([newIDString]);
  const [groupName, renamed] = await Text.asyncEdit(List.itemNameID(newIDString));
  if (!renamed) return false;
  return { ...data, group: { ...data.group, name: groupName } };
};

export const useCreateFromSelection = () => {
  const { update } = base.useUpdate({ beforeUpdate });
  return (props: Tree.ContextMenuProps) =>
    update({ ...props, group: { key: uuid.create(), name: "" } });
};
