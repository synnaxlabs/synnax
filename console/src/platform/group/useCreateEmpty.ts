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

const beforeUpdate = async ({ data }: Flux.BeforeUpdateParams<Group.CreateParams>) => {
  if (data.key == null) throw new Error("a group drafted in a tree needs a key");
  const [name, renamed] = await Text.asyncEdit(
    List.itemNameID(resource.idToString(group.resourceID(data.key))),
  );
  if (!renamed || name === "") return false;
  return { ...data, name };
};

/**
 * Returns a function that shows a draft group under the parent and creates the group
 * once the user names it.
 */
export const useCreateEmpty = ({
  parent,
  root,
  state: { addDraft, expand },
}: UseCreateEmptyProps): (() => void) => {
  const { updateAsync } = Group.useCreate({ beforeUpdate });
  return () => {
    const id = group.resourceID(uuid.create());
    const key = resource.idToString(id);
    const parentKey = resource.idToString(parent);
    if (!resource.idsEqual(parent, root)) expand(parentKey);
    const done = addDraft({ resource: { key, id, name: "" }, parent: parentKey });
    void updateAsync({ key: id.key, name: "", ...placement(parent, root) }).finally(
      done,
    );
  };
};
