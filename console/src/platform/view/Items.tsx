// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement } from "react";

import { useContext, useFormContext } from "@/platform/view/context";

export interface ItemsProps<K extends record.Key = record.Key>
  extends
    Omit<List.ScrollProps, "children" | "contextMenu">,
    Pick<List.ItemsProps<K>, "children" | "emptyContent"> {
  contextMenu?: Menu.ContextMenuProps["menu"];
}

export const Items = <K extends record.Key>({
  contextMenu,
  children,
  emptyContent,
  ...rest
}: ItemsProps<K>): ReactElement => {
  const menuProps = Menu.useContextMenu();
  const { answered } = useFormContext("View.Items");
  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <List.Scroll grow onContextMenu={menuProps.open} {...rest}>
        <List.Items<K> emptyContent={emptyContent ?? (answered && defaultEmptyContent)}>
          {children}
        </List.Items>
      </List.Scroll>
    </Menu.ContextMenu>
  );
};

const DefaultEmptyContent = (): ReactElement => {
  const { resourceType } = useContext("View.Items");
  return (
    <Flex.Box center>
      <Text.Text status="disabled">No {plural(resourceType)} found</Text.Text>
    </Flex.Box>
  );
};

const defaultEmptyContent = <DefaultEmptyContent />;
