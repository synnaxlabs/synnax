// Copyright 2026 Synnax Labs, Inc.
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
  Menu,
  Select,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { ContextMenu as CMenu } from "@/components";
import { CSS } from "@/css";
import { useIsSnapshot } from "@/hardware/common/task/Form";
import { type Channel } from "@/hardware/common/task/types";

export interface ContextMenuItemProps<C extends Channel> {
  keys: string[];
  channels: C[];
}

interface ContextMenuProps<C extends Channel> extends Pick<
  Form.UseFieldListReturn<C["key"], C>,
  "data" | "remove"
> {
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
  const isSnapshot = useIsSnapshot();
  const handleRemove = () => onSelect(array.toArray(remove(keys)[0]));
  const { get, set } = Form.useContext();
  const channels = Form.useFieldValue<C[]>(path).filter(({ key }) =>
    keys.includes(key),
  );
  const handleDuplicate = () => {
    const allChannels = get<C[]>(path).value;
    onDuplicate?.(allChannels, keys);
  };
  const handleDisable = () =>
    keys.forEach((key) => set(`${path}.${key}.enabled`, false));
  const handleEnable = () => keys.forEach((key) => set(`${path}.${key}.enabled`, true));
  const handleTare = useCallback(
    () => onTare?.(keys, channels),
    [onTare, keys, channels],
  );
  const canDuplicate = onDuplicate != null && keys.length > 0;
  const canRemove = keys.length > 0;
  const canDisable = channels.some(({ enabled }) => enabled);
  const canEnable = channels.some(({ enabled }) => !enabled);
  const canTare = allowTare?.(keys, channels) ?? false;
  return (
    <CMenu.Menu>
      {!isSnapshot && (
        <>
          {canDuplicate && (
            <Menu.Item itemKey="duplicate" onClick={handleDuplicate}>
              <Icon.Copy />
              Duplicate
            </Menu.Item>
          )}
          {canRemove && (
            <Menu.Item itemKey="remove" onClick={handleRemove}>
              <Icon.Close />
              Remove
            </Menu.Item>
          )}
          {(canDuplicate || canRemove) && <Menu.Divider />}
          {contextMenuItems?.({ channels, keys }) ?? null}
          {canDisable && (
            <Menu.Item itemKey="disable" onClick={handleDisable}>
              <Icon.Disable />
              Disable
            </Menu.Item>
          )}
          {canEnable && (
            <Menu.Item itemKey="enable" onClick={handleEnable}>
              <Icon.Enable />
              Enable
            </Menu.Item>
          )}
          {(canDisable || canEnable) && <Menu.Divider />}
          {canTare && (
            <>
              <Menu.Item itemKey="tare" onClick={handleTare}>
                <Icon.Tare />
                Tare
              </Menu.Item>
              <Menu.Divider />
            </>
          )}
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};

export interface ChannelListItemProps extends List.ItemProps<string> {}

export interface ChannelListProps<C extends Channel>
  extends
    Omit<ContextMenuProps<C>, "keys">,
    Pick<Flex.BoxProps, "onDragOver" | "onDrop" | "grow" | "style"> {
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
  style,
  ...rest
}: ChannelListProps<C>) => {
  const { onSelect, path, data } = rest;
  const handleChange = useCallback(
    (keys: string[]) => onSelect(keys),
    [onSelect, path],
  );
  const menuProps = Menu.useContextMenu();
  return (
    <Flex.Box className={CSS.B("channel-list")} empty grow={grow} style={style}>
      {header}
      <Menu.ContextMenu
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
      </Menu.ContextMenu>
    </Flex.Box>
  );
};
