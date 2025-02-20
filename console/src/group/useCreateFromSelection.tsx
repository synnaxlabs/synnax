// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { createNewID } from "@/group/createNewID";
import { getResourcesToGroup } from "@/group/getResourcesToGroup";
import { type Ontology } from "@/ontology";

export const useCreateFromSelection = () => {
  const mut = useMutation<
    void,
    Error,
    Ontology.TreeContextMenuProps & { newID: ontology.ID },
    Tree.Node[]
  >({
    onMutate: async ({
      selection,
      state: { nodes, setNodes, setSelection },
      newID,
    }) => {
      if (selection.parent == null) return;
      const resourcesToGroup = getResourcesToGroup(selection);
      const prevNodes = Tree.deepCopy(nodes);
      let nextNodes = Tree.setNode({
        tree: nodes,
        destination: selection.parent.key,
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
    mutationFn: async ({ client, selection, newID }) => {
      if (selection.parent == null) return;
      const [groupName, renamed] = await Tree.asyncRename(newID.toString());
      if (!renamed) throw errors.CANCELED;
      const resourcesToGroup = getResourcesToGroup(selection);
      const parentID = new ontology.ID(selection.parent.key);
      await client.ontology.groups.create(parentID, groupName, newID.key);
      await client.ontology.moveChildren(parentID, newID, ...resourcesToGroup);
    },
    onError: async (e, { state: { setNodes }, handleException }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(e.message)) return;
      handleException(e, "Failed to group resources");
    },
  });
  return (props: Ontology.TreeContextMenuProps) =>
    mut.mutate({ ...props, newID: createNewID() });
};
