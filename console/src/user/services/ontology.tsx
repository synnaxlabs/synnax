// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { type Flux, Icon, Menu as PMenu, Text, User } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Menu } from "@/components";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";

const useDelete = createUseDelete({
  type: "User",
  query: User.useDelete,
  convertKey: String,
});

const useRename = ({
  selection: {
    ids: [firstID],
  },
  state: { getResource },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const beforeUpdate = useCallback(
    async ({ data }: Flux.BeforeUpdateParams<User.ChangeUsernameParams>) => {
      const [username, renamed] = await Text.asyncEdit(ontology.idToString(firstID));
      if (!renamed) return false;
      return { ...data, username };
    },
    [firstID],
  );
  const { update } = User.useRename({ beforeUpdate });
  return useCallback(
    () => update({ key: firstID.key, username: getResource(firstID).name }),
    [update, firstID, getResource],
  );
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    client,
    state: { getResource },
    selection: { ids },
  } = props;
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const handleSelect = {
    rename,
    delete: handleDelete,
  };
  const singleResource = ids.length === 1;
  const isNotCurrentUser = getResource(ids[0]).name !== client.params.username;

  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      {singleResource && isNotCurrentUser && (
        <>
          <PMenu.Item itemKey="rename">
            <Icon.Rename />
            Change username
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <>
        <Menu.DeleteItem />
        <PMenu.Divider />
      </>
      {singleResource && (
        <>
          <Ontology.CopyMenuItem {...props} />
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "user",
  icon: <Icon.User />,
  TreeContextMenu,
};
