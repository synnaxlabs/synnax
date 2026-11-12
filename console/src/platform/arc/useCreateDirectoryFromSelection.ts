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
import { Arc, Flux } from "@synnaxlabs/pluto";
import { uuid, verbs } from "@synnaxlabs/x";

import { type Tree } from "@/platform/tree";

interface CreateParams extends Tree.ContextMenuProps {
  directory: { key: arc.directory.Key; name: string };
}

const base = Flux.createUpdate<CreateParams>({
  name: Arc.Directory.RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    const {
      selection: { parentID, top },
      directory: { key, name },
    } = data;
    if (parentID == null)
      throw new Error("Cannot move items that sit in more than one directory");
    const { directories } = client.arcs;
    await client.batch.send([
      directories.createRequest({ parent: parentID.key, key, name }),
      ...directories.moveRequests(arc.directory.resourceID(key), ...top),
    ]);
    return data;
  },
});

const beforeUpdate = async ({ data }: Flux.BeforeUpdateParams<CreateParams>) => {
  const key = resource.idToString(arc.directory.resourceID(data.directory.key));
  const [name, renamed] = await Text.asyncEdit(List.itemNameID(key));
  if (!renamed || name === "") return false;
  return { ...data, directory: { ...data.directory, name } };
};

/**
 * Returns a function that shows a draft directory holding the top of the selection,
 * where the selection shows, and creates the directory once the user names it.
 */
export const useCreateDirectoryFromSelection = () => {
  const { updateAsync } = base.useUpdate({ beforeUpdate });
  return (props: Tree.ContextMenuProps) => {
    const {
      selection: { top },
      state: { addDraft, getParent, setSelection },
    } = props;
    const directoryKey = uuid.create();
    const id = arc.directory.resourceID(directoryKey);
    const key = resource.idToString(id);
    const children = resource.idToString(top);
    const parent = getParent(children[0]);
    if (parent == null) throw new Error(`${children[0]} does not show in the tree`);
    const done = addDraft({ resource: { key, id, name: "" }, parent, children });
    setSelection([key]);
    void (async () => {
      await updateAsync({ ...props, directory: { key: directoryKey, name: "" } });
      done();
    })();
  };
};
