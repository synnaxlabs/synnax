// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Tree } from "@synnaxlabs/pluto";
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
      state: { nodes, setNodes, setSelection },
      newID,
    }) => {
      if (selection.parentID == null) return;
      const resourcesToGroup = getResourcesToGroup(selection);
      const prevNodes = Tree.deepCopy(nodes);
      let nextNodes = Tree.setNode({
        tree: nodes,
        destination:
          selection.rootID.toString() == selection.parentID.toString()
            ? null
            : selection.parentID.toString(),
        additions: {
          key: newID.toString(),
          icon: <Icon.Group />,
          children: [],
          name: "",
          allowRename: true,
        },
      });
      nextNodes = Tree.moveNode({
        tree: nextNodes,
        destination: newID.toString(),
        keys: resourcesToGroup.map((id) => id.toString()),
      });
      setNodes([...nextNodes]);
      setSelection([newID.toString()]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection, newID }: CreateArgs) => {
      if (selection.parentID == null) return;
      const [groupName, renamed] = await Tree.asyncRename(newID.toString());
      if (!renamed) throw new errors.Canceled();
      const resourcesToGroup = getResourcesToGroup(selection);
      const parentID = new ontology.ID(selection.parentID.toString());
      await client.ontology.groups.create(parentID, groupName, newID.key);
      await client.ontology.moveChildren(parentID, newID, ...resourcesToGroup);
    },
    onError: async (e, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e.message)) return;
      handleError(e, "Failed to group resources");
    },
  }).mutate;
  return useCallback(
    (props) => create({ ...props, newID: group.ontologyID(uuid.create()) }),
    [create],
  );
};
