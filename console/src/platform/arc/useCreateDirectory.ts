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
import { Arc, type Flux } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { type Tree } from "@/platform/tree";

export interface UseCreateDirectoryProps {
  /** The directory to create the new directory in. */
  parent: resource.ID;
  root: resource.ID;
  state: Tree.TreeState;
}

const beforeUpdate = async ({
  data,
}: Flux.BeforeUpdateParams<Arc.Directory.CreateParams>) => {
  if (data.key == null) throw new Error("a directory drafted in a tree needs a key");
  const [name, renamed] = await Text.asyncEdit(
    List.itemNameID(resource.idToString(arc.directory.resourceID(data.key))),
  );
  if (!renamed || name === "") return false;
  return { ...data, name };
};

/**
 * Returns a function that shows a draft directory under parent and creates the
 * directory once the user names it.
 */
export const useCreateDirectory = ({
  parent,
  root,
  state: { addDraft, expand },
}: UseCreateDirectoryProps): (() => void) => {
  const { updateAsync } = Arc.Directory.useCreate({ beforeUpdate });
  return () => {
    const id = arc.directory.resourceID(uuid.create());
    const key = resource.idToString(id);
    const parentKey = resource.idToString(parent);
    if (!resource.idsEqual(parent, root)) expand(parentKey);
    const done = addDraft({ resource: { key, id, name: "" }, parent: parentKey });
    void (async () => {
      await updateAsync({ parent: parent.key, key: id.key, name: "" });
      done();
    })();
  };
};
