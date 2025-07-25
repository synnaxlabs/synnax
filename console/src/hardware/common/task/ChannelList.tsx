// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Align,
  Form,
  Icon,
  List,
  Menu as PMenu,
  type RenderProp,
  Select,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { type Channel } from "@/hardware/common/task/types";

export interface ContextMenuItemProps<C extends Channel> {
  keys: string[];
  channels: C[];
}

interface ContextMenuProps<C extends Channel>
  extends Pick<Form.UseFieldListReturn<C["key"], C>, "data" | "remove"> {
  keys: string[];
  allowTare?: (keys: string[], channels: C[]) => boolean;
  isSnapshot: boolean;
  onDuplicate?: (channels: C[], keys: string[]) => void;
  onSelect: (keys: string[]) => void;
  onTare?: (keys: string[], channels: C[]) => void;
  path: string;
  contextMenuItems?: RenderProp<ContextMenuItemProps<C>>;
}

const ContextMenu = <C extends Channel>({
  allowTare,
  keys,
  isSnapshot,
  onDuplicate,
  onSelect,
  onTare,
  path,
  remove,
  contextMenuItems,
}: ContextMenuProps<C>) => {
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
            <PMenu.Item itemKey="duplicate" startIcon={<Icon.Copy />}>
              Duplicate
            </PMenu.Item>
          )}
          {canRemove && (
            <>
              <PMenu.Item itemKey="remove" startIcon={<Icon.Close />}>
                Remove
              </PMenu.Item>
            </>
          )}
          {(canDuplicate || canRemove) && <PMenu.Divider />}
          {contextMenuItems?.({ channels, keys }) ?? null}
          {canDisable && (
            <PMenu.Item itemKey="disable" startIcon={<Icon.Disable />}>
              Disable
            </PMenu.Item>
          )}
          {canEnable && (
            <PMenu.Item itemKey="enable" startIcon={<Icon.Enable />}>
              Enable
            </PMenu.Item>
          )}
          {(canDisable || canEnable) && <PMenu.Divider />}
          {canTare && (
            <>
              <PMenu.Item itemKey="tare" startIcon={<Icon.Tare />}>
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

export interface ChannelListItemProps extends List.ItemProps<string> {
  key: string;
  isSnapshot: boolean;
  path: string;
}

export interface ChannelListProps<C extends Channel>
  extends Omit<ContextMenuProps<C>, "keys">,
    Pick<Align.SpaceProps, "onDragOver" | "onDrop" | "grow"> {
  emptyContent: ReactElement;
  header: ReactNode;
  isDragging?: boolean;
  listItem: RenderProp<ChannelListItemProps>;
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
  const { isSnapshot, onSelect, path, data } = rest;
  const handleChange = useCallback(
    (keys: string[]) => onSelect(keys),
    [onSelect, path],
  );
  const menuProps = PMenu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channel-list")} empty grow={grow}>
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
        >
          <List.Items<string, C>
            onDragOver={onDragOver}
            onDrop={onDrop}
            className={menuProps.className}
            onContextMenu={menuProps.open}
            emptyContent={emptyContent}
          >
            {(props) =>
              listItem({ isSnapshot, path: `${path}.${props.key}`, ...props })
            }
          </List.Items>
        </Select.Frame>
      </PMenu.ContextMenu>
    </Align.Space>
  );
};
