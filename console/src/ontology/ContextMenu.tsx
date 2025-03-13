// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { Group } from "@/group";
import { type TreeContextMenu } from "@/ontology/service";

export const MultipleSelectionContextMenu: TreeContextMenu = (props) => {
  const group = Group.useCreateFromSelection();
  const handleSelect = {
    group: () => group(props),
  };
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.MenuItem selection={props.selection} />
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};
