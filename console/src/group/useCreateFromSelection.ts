// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Flux, Group, Text, Tree } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { getResourcesToGroup } from "@/group/getResourcesToGroup";
import { type Ontology } from "@/ontology";

export interface CreateFromSelection {
  (props: Ontology.TreeContextMenuProps): void;
}

interface CreateParams extends Ontology.TreeContextMenuProps {
  group: group.Group;
  prevNodes?: Tree.Node<string>[];
}

const base = Flux.createUpdate<CreateParams, Group.FluxSubStore>({
  name: Group.RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data }) => {
    const {
      selection: { parentID, ids },
      state: { shape },
      group: { name, key },
    } = data;
    const resourcesToGroup = getResourcesToGroup(ids, shape);
    await client.ontology.groups.create({ parent: parentID, name, key });
    await client.ontology.moveChildren(
      parentID,
      group.ontologyID(key),
      ...resourcesToGroup,
    );
    return data;
  },
});

const beforeUpdate = async ({ data }: Flux.BeforeUpdateParams<CreateParams>) => {
  const {
    selection,
    state: { nodes, setNodes, setSelection, shape, setResource },
    group: { key },
  } = data;
  if (selection.parentID == null) return false;
  const newID = group.ontologyID(key);
  const newIDString = ontology.idToString(newID);
  const resourcesToGroup = getResourcesToGroup(selection.ids, shape);
  const prevNodes = Tree.deepCopy(nodes);
  const res: ontology.Resource = { key: newIDString, id: newID, name: "" };
  setResource(res);
  const destination = ontology.idsEqual(selection.rootID, selection.parentID)
    ? null
    : ontology.idToString(selection.parentID);
  let nextNodes = Tree.setNode({
    tree: nodes,
    destination,
    additions: { key: ontology.idToString(newID), children: [] },
  });
  nextNodes = Tree.moveNode({
    tree: nextNodes,
    destination: ontology.idToString(newID),
    keys: resourcesToGroup.map((id) => ontology.idToString(id)),
  });
  setNodes([...nextNodes]);
  setSelection([ontology.idToString(newID)]);
  const [groupName, renamed] = await Text.asyncEdit(ontology.idToString(newID));
  if (!renamed) {
    setNodes(prevNodes);
    return false;
  }

  return { ...data, prevNodes, group: { ...data.group, name: groupName } };
};

const afterFailure = async ({
  status,
  data: {
    prevNodes,
    addStatus,
    state: { setNodes },
  },
}: Flux.AfterFailureParams<CreateParams>) => {
  if (prevNodes != null) setNodes(prevNodes);
  addStatus(status);
};
export const useCreateFromSelection = () => {
  const { update } = base.useUpdate({ beforeUpdate, afterFailure });
  return useCallback(
    (props: Ontology.TreeContextMenuProps) =>
      update({ ...props, group: { key: uuid.create(), name: "" } }),
    [update],
  );
};
