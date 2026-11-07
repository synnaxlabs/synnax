// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, resource } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Arc, type Flux } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { type Tree } from "@/platform/tree";

export interface UseCreateDirectoryProps {
  parent: resource.ID;
  root: resource.ID;
  state: Tree.TreeState;
}

/**
 * Returns a function that adds an unnamed directory row under parent and creates the
 * directory once the user names it.
 */
export const useCreateDirectory = ({
  parent,
  root,
  state: { nodes: tree, setNodes, setResource, expand },
}: UseCreateDirectoryProps): (() => void) => {
  const { update } = Arc.Directory.useCreate({
    beforeUpdate: async ({
      data,
      rollbacks,
    }: Flux.BeforeUpdateParams<Arc.Directory.CreateParams>) => {
      const newID = arc.directory.resourceID(uuid.create());
      const newIDString = resource.idToString(newID);
      const res: Tree.Resource = { key: newIDString, id: newID, name: "" };
      const node: PTree.Node<string> = { key: newIDString, children: [] };
      setResource(res);
      const destination = resource.idsEqual(parent, root)
        ? null
        : resource.idToString(parent);
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
  return () => update({ parent: parent.key, name: "" });
};
