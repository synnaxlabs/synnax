// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type ontology, resource } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { type Flux, Group } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { placement } from "@/platform/group/placement";
import { type Tree } from "@/platform/tree";

export interface UseCreateEmptyProps {
  /** The node to create the group under. */
  parent: resource.ID;
  root: resource.ID;
  state: Tree.TreeState;
}

export const useCreateEmpty = ({
  parent,
  root,
  state: { nodes: tree, setNodes, setResource, expand },
}: UseCreateEmptyProps): (() => void) => {
  const { update } = Group.useCreate({
    beforeUpdate: async ({
      data,
      rollbacks,
    }: Flux.BeforeUpdateParams<Group.CreateParams>) => {
      const newID = group.resourceID(uuid.create());
      const newIDString = resource.idToString(newID);
      const res: ontology.Resource = { key: newIDString, id: newID, name: "" };
      const node: PTree.Node<string> = { key: newIDString, children: [] };
      setResource(res);
      const destination = data.parent == null ? null : resource.idToString(data.parent);
      if (destination != null) expand(destination);
      setNodes([...PTree.setNode({ tree, destination, additions: node })]);
      rollbacks.push(() =>
        setNodes([...PTree.removeNode({ tree, keys: newIDString })]),
      );
      const [name, renamed] = await Text.asyncEdit(List.itemNameID(newIDString));
      if (!renamed || name === "") return false;
      return { ...data, key: newID.key, name };
    },
  });
  return () => update({ key: uuid.create(), name: "", ...placement(parent, root) });
};
