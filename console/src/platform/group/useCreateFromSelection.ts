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
import { Flux, Group } from "@synnaxlabs/pluto";
import { uuid, verbs } from "@synnaxlabs/x";

import { placement } from "@/platform/group/placement";
import { type Tree } from "@/platform/tree";

interface CreateParams extends Tree.ContextMenuProps {
  group: Pick<group.Group, "key" | "name">;
}

const base = Flux.createUpdate<CreateParams>({
  name: Group.RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => {
    const {
      selection: { parentID, rootID, top },
      group: { name, key },
    } = data;
    if (parentID == null)
      throw new Error("Cannot group items that sit under more than one parent");
    await client.batch.send([
      client.groups.createRequest({ key, name, ...placement(parentID, rootID) }),
      ...client.groups.moveRequests(top, group.resourceID(key)),
    ]);
    return data;
  },
});

const beforeUpdate = async ({ data }: Flux.BeforeUpdateParams<CreateParams>) => {
  const key = resource.idToString(group.resourceID(data.group.key));
  const [name, renamed] = await Text.asyncEdit(List.itemNameID(key));
  if (!renamed) return false;
  return { ...data, group: { ...data.group, name } };
};

/**
 * Returns a function that shows a draft group holding the top of the selection, where
 * the selection shows, and creates the group once the user names it.
 */
export const useCreateFromSelection = () => {
  const { updateAsync } = base.useUpdate({ beforeUpdate });
  return (props: Tree.ContextMenuProps) => {
    const {
      selection: { top },
      state: { addDraft, getParent, setSelection },
    } = props;
    const groupKey = uuid.create();
    const id = group.resourceID(groupKey);
    const key = resource.idToString(id);
    const children = resource.idToString(top);
    // In flat search results the selection shows at the top of the tree, while its real
    // parent is a group.
    const parent = getParent(children[0]);
    if (parent == null) throw new Error(`${children[0]} does not show in the tree`);
    const done = addDraft({ resource: { key, id, name: "" }, parent, children });
    setSelection([key]);
    void (async () => {
      await updateAsync({ ...props, group: { key: groupKey, name: "" } });
      done();
    })();
  };
};
