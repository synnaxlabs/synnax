// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";
import { type Flux, Group, List, Text, Tree as PTree } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type Tree } from "@/platform/tree";

export interface UseCreateEmptyProps {
  parent: ontology.ID;
  root: ontology.ID;
  state: Tree.TreeState;
}

export const useCreateEmpty = ({
  parent,
  root,
  state: { nodes: tree, setNodes, setResource, expand },
}: UseCreateEmptyProps): (() => void) => {
  const { update } = Group.useCreate({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Group.CreateParams>) => {
        const newID = group.ontologyID(uuid.create());
        const newIDString = ontology.idToString(newID);
        const res: ontology.Resource = { key: newIDString, id: newID, name: "" };
        const node: PTree.Node<string> = { key: newIDString, children: [] };
        setResource(res);
        const destination = ontology.idsEqual(data.parent, root)
          ? null
          : ontology.idToString(data.parent);
        if (destination != null) expand(destination);
        setNodes([...PTree.setNode({ tree, destination, additions: node })]);
        rollbacks.push(() =>
          setNodes([...PTree.removeNode({ tree, keys: newIDString })]),
        );
        const [name, renamed] = await Text.asyncEdit(List.itemNameID(newIDString));
        if (!renamed || name === "") return false;
        return { ...data, key: newID.key, name };
      },
      [tree, setNodes, setResource, expand],
    ),
  });
  return useCallback(
    () => update({ key: uuid.create(), name: "", parent }),
    [parent, update],
  );
};
