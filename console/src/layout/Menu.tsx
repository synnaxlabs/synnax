// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon, Menu, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu } from "@/components/context-menu";
import { setFocus } from "@/layout/slice";
import { useRemover } from "@/layout/useRemover";

interface MenuItemProps {
  layoutKey: string;
}

const FocusMenuItem = ({ layoutKey }: MenuItemProps): ReactElement => {
  const dispatch = useDispatch();
  const windowKey = useSelectWindowKey() as string;
  return (
    <Menu.Item
      itemKey="focus"
      onClick={() => dispatch(setFocus({ windowKey, key: layoutKey }))}
      trigger={["Control", "L"]}
    >
      <Icon.Focus />
      Focus
    </Menu.Item>
  );
};

const CloseMenuItem = ({ layoutKey }: MenuItemProps): ReactElement => {
  const remove = useRemover();
  return (
    <Menu.Item
      itemKey="close"
      onClick={() => remove(layoutKey)}
      trigger={["Control", "W"]}
      triggerIndicator
    >
      <Icon.Close />
      Close
    </Menu.Item>
  );
};

const RenameMenuItem = ({ layoutKey }: MenuItemProps): ReactElement => (
  <ContextMenu.RenameItem
    onClick={() => Text.edit(`pluto-tab-${layoutKey}`)}
    trigger={["Control", "E"]}
    triggerIndicator
  />
);

export interface MenuItemsProps {
  layoutKey: string;
}

// MenuItems renders the shared layout context-menu items. Split and multi-window
// items (split node, open/move to window) were mosaic operations; on the panel
// model splitting is a tab-strip gesture and multi-window is rebuilt in a later
// phase, so they are not included here.
export const MenuItems = ({ layoutKey }: MenuItemsProps): ReactElement => (
  <>
    <RenameMenuItem layoutKey={layoutKey} />
    <CloseMenuItem layoutKey={layoutKey} />
    <Menu.Divider />
    <FocusMenuItem layoutKey={layoutKey} />
    <Menu.Divider />
    <ContextMenu.ReloadConsoleItem />
  </>
);
