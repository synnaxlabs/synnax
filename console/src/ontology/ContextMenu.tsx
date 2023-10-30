// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";

import { Group } from "@/group";
import { type TreeContextMenu } from "@/ontology/service";

export const MultipleSelectionContextMenu: TreeContextMenu = (props) => {
  const handleSelect: Menu.MenuProps["onChange"] = (key) => {
    switch (key) {
      case "group":
        void Group.fromSelection(props);
    }
  };

  return (
    <Menu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.GroupMenuItem selection={props.selection} />
    </Menu.Menu>
  );
};

export const RenameMenuItem = (): ReactElement => (
  <Menu.Item itemKey="rename" startIcon={<Icon.Rename />}>
    Rename
  </Menu.Item>
);
