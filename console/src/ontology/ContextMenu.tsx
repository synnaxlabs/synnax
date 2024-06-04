// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as pMenu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Group } from "@/group";
import { Menu } from "@/menu";
import { TreeContextMenuProps, type TreeContextMenu } from "@/ontology/service";

export interface MultipleSelectionContextMenuProps extends TreeContextMenuProps {
  canGroup: boolean;
}

export const MultipleSelectionContextMenu: TreeContextMenu = (props) => {
  const handleSelect: pMenu.MenuProps["onChange"] = (key) => {
    switch (key) {
      case "group":
        void Group.fromSelection(props);
    }
  };

  return (
    <pMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.GroupMenuItem selection={props.selection} />
      <Menu.HardReload />
    </pMenu.Menu>
  );
};

export const RenameMenuItem = (): ReactElement => (
  <pMenu.Item itemKey="rename" startIcon={<Icon.Rename />}>
    Rename
  </pMenu.Item>
);
