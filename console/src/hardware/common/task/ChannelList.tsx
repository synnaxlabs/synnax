// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Component,
  Flex,
  Form,
  Icon,
  List,
  Menu as PMenu,
  Select,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { type Channel } from "@/hardware/common/task/types";

export interface ContextMenuItemProps<C extends Channel> {
  keys: string[];
  channels: C[];
}

interface ContextMenuProps<C extends Channel>
  extends Pick<Form.UseFieldListReturn<C["key"], C>, "data" | "remove"> {
  keys: string[];
  allowTare?: (keys: string[], channels: C[]) => boolean;
  onDuplicate?: (channels: C[], keys: string[]) => void;
  onSelect: (keys: string[]) => void;
  onTare?: (keys: string[], channels: C[]) => void;
  path: string;
  contextMenuItems?: Component.RenderProp<ContextMenuItemProps<C>>;
}

const ContextMenu = <C extends Channel>({
  allowTare,
  keys,
  onDuplicate,
  onSelect,
  onTare,
  path,
  remove,
  contextMenuItems,
}: ContextMenuProps<C>) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  const handleRemove = () => onSelect(array.toArray(remove(keys)[0]));
  const { set } = Form.useContext();
  const channels = Form.useFieldValue<C[]>(path).filter(({ key }) =>
    keys.includes(key),
  );
  const handleDuplicate = () => onDuplicate?.(channels, keys);
  const handleDisable = () =>
    keys.forEach((key) => set(`${path}.${key}.enabled`, false));
  const handleEnable = () => keys.forEach((key) => set(`${path}.${key}.enabled`, true));
  const handleTare = useCallback(
    () => onTare?.(keys, channels),
    [onTare, keys, channels],
  );
  const handleSelect: Record<string, () => void> = {
    remove: handleRemove,
    disable: handleDisable,
    enable: handleEnable,
    tare: handleTare,
    duplicate: handleDuplicate,
  };
  const canDuplicate = onDuplicate != null && keys.length > 0;
  const canRemove = keys.length > 0;
  const canDisable = channels.some(({ enabled }) => enabled);
  const canEnable = channels.some(({ enabled }) => !enabled);
  const canTare = allowTare?.(keys, channels) ?? false;
  return (
    <PMenu.Menu onChange={handleSelect} level="small">
      {!isSnapshot && (
        <>
          {canDuplicate && (
            <PMenu.Item itemKey="duplicate">
              <Icon.Copy />
              Duplicate
            </PMenu.Item>
          )}
          {canRemove && (
            <>
              <PMenu.Item itemKey="remove">
                <Icon.Close />
                Remove
              </PMenu.Item>
            </>
          )}
          {(canDuplicate || canRemove) && <PMenu.Divider />}
          {contextMenuItems?.({ channels, keys }) ?? null}
          {canDisable && (
            <PMenu.Item itemKey="disable">
              <Icon.Disable />
              Disable
            </PMenu.Item>
          )}
          {canEnable && (
            <PMenu.Item itemKey="enable">
              <Icon.Enable />
              Enable
            </PMenu.Item>
          )}
          {(canDisable || canEnable) && <PMenu.Divider />}
          {canTare && (
            <>
              <PMenu.Item itemKey="tare">
                <Icon.Tare />
                Tare
              </PMenu.Item>
              <PMenu.Divider />
            </>
          )}
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export interface ChannelListItemProps extends List.ItemProps<string> {}

export interface ChannelListProps<C extends Channel>
  extends Omit<ContextMenuProps<C>, "keys">,
    Pick<Flex.BoxProps, "onDragOver" | "onDrop" | "grow"> {
  emptyContent: ReactElement;
  header: ReactNode;
  isDragging?: boolean;
  listItem: Component.RenderProp<ChannelListItemProps>;
  selected: string[];
}

export const ChannelList = <C extends Channel>({
  listItem,
  emptyContent,
  header,
  isDragging,
  onDragOver,
  onDrop,
  selected,
  grow,
  ...rest
}: ChannelListProps<C>) => {
  const { onSelect, path, data } = rest;
  const handleChange = useCallback(
    (keys: string[]) => onSelect(keys),
    [onSelect, path],
  );
  const menuProps = PMenu.useContextMenu();
  return (
    <Flex.Box className={CSS.B("channel-list")} empty grow={grow}>
      {header}
      <PMenu.ContextMenu
        {...menuProps}
        menu={(p) => <ContextMenu {...p} {...rest} />}
        onDragOver={onDragOver}
        onDrop={onDrop}
      >
        <Select.Frame<string, C>
          multiple
          data={data}
          value={selected}
          onChange={handleChange}
          replaceOnSingle
          allowNone={false}
          autoSelectOnNone
        >
          <List.Items<string, C>
            full="y"
            onDragOver={onDragOver}
            onDrop={onDrop}
            className={menuProps.className}
            onContextMenu={menuProps.open}
            emptyContent={emptyContent}
          >
            {listItem}
          </List.Items>
        </Select.Frame>
      </PMenu.ContextMenu>
    </Flex.Box>
  );
};
