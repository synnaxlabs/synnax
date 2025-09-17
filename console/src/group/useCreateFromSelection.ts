// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Flux, Text, Tree } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { getResourcesToGroup } from "@/group/getResourcesToGroup";
import { type Ontology } from "@/ontology";

export interface CreateFromSelection {
  (props: Ontology.TreeContextMenuProps): void;
}

interface CreateArgs extends Ontology.TreeContextMenuProps {
  group: group.Group;
  prevNodes?: Tree.Node<string>[];
}

export const useCreateFromSelection = () => {
  const { update } = Flux.createUpdate<CreateArgs>({
    name: "Group",
    update: async ({ client, value }) => {
      const {
        selection: { parentID, ids },
        state: { shape },
        group: { name, key },
      } = value;
      const resourcesToGroup = getResourcesToGroup(ids, shape);
      await client.ontology.groups.create({ parent: parentID, name, key });
      await client.ontology.moveChildren(
        parentID,
        group.ontologyID(key),
        ...resourcesToGroup,
      );
      return value;
    },
  }).useUpdate({
    beforeUpdate: async ({ value }) => {
      const {
        selection,
        state: { nodes, setNodes, setSelection, shape, setResource },
        group: { key },
      } = value;
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
      if (!renamed) return false;
      return { ...value, prevNodes, group: { ...value.group, name: groupName } };
    },
    afterFailure: async ({
      status,
      value: {
        prevNodes,
        addStatus,
        state: { setNodes },
      },
    }) => {
      if (prevNodes != null) setNodes(prevNodes);
      addStatus(status);
    },
  });
  return useCallback(
    (props: Ontology.TreeContextMenuProps) =>
      update({ ...props, group: { key: uuid.create(), name: "" } }),
    [update],
  );
};
