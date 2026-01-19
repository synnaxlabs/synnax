// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon, Menu as PMenu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Menu } from "@/components";
import { Group } from "@/group";
import { type TreeState } from "@/ontology/service";

export interface DefaultContextMenuProps {
  root: ontology.ID;
  state: TreeState;
}

export const DefaultContextMenu = ({
  root,
  state,
}: DefaultContextMenuProps): ReactElement => {
  const createGroup = Group.useCreateEmpty({ parent: root, state, root });
  const handleSelect = { newGroup: createGroup };
  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      <PMenu.Item itemKey="newGroup">
        <Icon.Group />
        New Group
      </PMenu.Item>
      <PMenu.Divider />
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};
