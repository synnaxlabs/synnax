// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Text, Tree } from "@synnaxlabs/pluto";
import { errors, uuid } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";

import { getResourcesToGroup } from "@/group/getResourcesToGroup";
import { type Ontology } from "@/ontology";

export interface CreateFromSelection {
  (props: Ontology.TreeContextMenuProps): void;
}

interface CreateArgs extends Ontology.TreeContextMenuProps {
  newID: ontology.ID;
}

export const useCreateFromSelection = (): CreateFromSelection => {
  const create = useMutation<void, Error, CreateArgs, Tree.Node[]>({
    onMutate: async ({
      selection,
      state: { nodes, setNodes, setSelection, shape, setResource },
      newID,
    }) => {
      if (selection.parentID == null) return;
      const resourcesToGroup = getResourcesToGroup(selection.resourceIDs, shape);
      const prevNodes = Tree.deepCopy(nodes);
      const res: ontology.Resource = {
        key: ontology.idToString(newID),
        id: newID,
        name: "",
      };
      setResource(res);
      const destination = ontology.idsEqual(selection.rootID, selection.parentID)
        ? null
        : ontology.idToString(selection.parentID);
      let nextNodes = Tree.setNode({
        tree: nodes,
        destination,
        additions: {
          key: ontology.idToString(newID),
          children: [],
        },
      });
      nextNodes = Tree.moveNode({
        tree: nextNodes,
        destination: ontology.idToString(newID),
        keys: resourcesToGroup.map((id) => ontology.idToString(id)),
      });
      setNodes([...nextNodes]);
      setSelection([ontology.idToString(newID)]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection, newID, state: { shape } }: CreateArgs) => {
      if (selection.parentID == null) return;
      const [groupName, renamed] = await Text.asyncEdit(ontology.idToString(newID));
      if (!renamed) throw new errors.Canceled();
      const resourcesToGroup = getResourcesToGroup(selection.resourceIDs, shape);
      await client.ontology.groups.create(selection.parentID, groupName, newID.key);
      await client.ontology.moveChildren(
        selection.parentID,
        newID,
        ...resourcesToGroup,
      );
    },
    onError: async (e, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to group resources");
    },
  }).mutate;
  return useCallback(
    (props) => create({ ...props, newID: group.ontologyID(uuid.create()) }),
    [create],
  );
};
