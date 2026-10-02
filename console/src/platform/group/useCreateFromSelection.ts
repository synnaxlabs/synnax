// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Flux, Group } from "@synnaxlabs/pluto";
import { uuid, verbs } from "@synnaxlabs/x";
import { useCallback } from "react";
import { flushSync } from "react-dom";

import { getResourcesToGroup } from "@/platform/group/getResourcesToGroup";
import { type Tree } from "@/platform/tree";

interface CreateParams extends Tree.ContextMenuProps {
  group: group.Group;
}

const base = Flux.createUpdate<CreateParams>({
  name: Group.RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    const {
      selection: { parentID, ids },
      state: { shape },
      group: { name, key },
    } = data;
    const resourcesToGroup = getResourcesToGroup(ids, shape);
    await client.groups.create({ parent: parentID, name, key });
    await client.ontology.moveChildren(
      parentID,
      group.ontologyID(key),
      ...resourcesToGroup,
    );
    return data;
  },
});

const beforeUpdate = async ({
  data,
  rollbacks,
}: Flux.BeforeUpdateParams<CreateParams>) => {
  const {
    selection,
    state: { nodes, setNodes, setSelection, shape, setResource, scrollTo },
    group: { key },
  } = data;
  const newID = group.ontologyID(key);
  const newIDString = ontology.idToString(newID);
  const resourcesToGroup = getResourcesToGroup(selection.ids, shape);
  const prevNodes = PTree.deepCopy(nodes);
  rollbacks.push(() => setNodes(prevNodes));
  const res: ontology.Resource = { key: newIDString, id: newID, name: "" };
  setResource(res);
  const destination = ontology.idsEqual(selection.rootID, selection.parentID)
    ? null
    : ontology.idToString(selection.parentID);
  let nextNodes = PTree.setNode({
    tree: nodes,
    destination,
    additions: { key: newIDString, children: [] },
  });
  nextNodes = PTree.moveNode({
    tree: nextNodes,
    destination: newIDString,
    keys: resourcesToGroup.map((id) => ontology.idToString(id)),
  });
  // The group sorts to the top of its parent, which may be outside the mounted rows.
  flushSync(() => setNodes([...nextNodes]));
  scrollTo(newIDString);
  setSelection([newIDString]);
  const [groupName, renamed] = await Text.asyncEdit(List.itemNameID(newIDString));
  if (!renamed) return false;
  return { ...data, group: { ...data.group, name: groupName } };
};

export const useCreateFromSelection = () => {
  const { update } = base.useUpdate({ beforeUpdate });
  return useCallback(
    (props: Tree.ContextMenuProps) =>
      update({ ...props, group: { key: uuid.create(), name: "" } }),
    [update],
  );
};
