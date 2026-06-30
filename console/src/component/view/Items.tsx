// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, List, Menu, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement } from "react";

import { useContext } from "@/view/context";

export interface ItemsProps<K extends record.Key = record.Key> extends Omit<
  List.ItemsProps<K>,
  "contextMenu"
> {
  contextMenu?: Menu.ContextMenuProps["menu"];
}

export const Items = <K extends record.Key>({
  contextMenu,
  ...props
}: ItemsProps<K>): ReactElement => {
  const menuProps = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <List.Items<K>
        emptyContent={defaultEmptyContent}
        grow
        onContextMenu={menuProps.open}
        {...props}
      />
    </Menu.ContextMenu>
  );
};

const DefaultEmptyContent = (): ReactElement => {
  const { resourceType } = useContext("View.Items");
  return (
    <Flex.Box center>
      <Text.Text status="disabled">No {plural(resourceType)} found.</Text.Text>
    </Flex.Box>
  );
};

const defaultEmptyContent = <DefaultEmptyContent />;
