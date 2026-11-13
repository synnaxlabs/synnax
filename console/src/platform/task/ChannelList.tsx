// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { array, deep } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, type UIEvent, useRef } from "react";

import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { BindChannels, type BindChannelsProps } from "@/platform/task/BindChannels";
import { TareItem } from "@/platform/task/TareItem";
import { type Channel } from "@/platform/task/types";
import { useIsPreview } from "@/platform/task/useIsPreview";

export interface ContextMenuItemProps<C extends Channel> {
  keys: string[];
  channels: C[];
}

interface ContextMenuProps<C extends Channel> extends Pick<
  Form.UseFieldListReturn<C>,
  "data" | "remove"
> {
  keys: string[];
  onDuplicate?: (channels: C[], keys: string[]) => void;
  onSelect: (keys: string[]) => void;
  /** The list's float read channels can tare while the task runs. */
  tareable?: boolean;
  path: deep.Path<C[]>;
  contextMenuItems?: Component.RenderProp<ContextMenuItemProps<C>>;
}

/** @returns the Synnax channel a read channel binds, or 0 for any other channel. */
const boundChannel = (c: Channel): channel.Key =>
  "channel" in c && typeof c.channel === "number" ? c.channel : 0;

const ContextMenu = <C extends Channel>({
  keys,
  onDuplicate,
  onSelect,
  tareable = false,
  path,
  remove,
  contextMenuItems,
}: ContextMenuProps<C>) => {
  const isPreview = useIsPreview();
  const handleRemove = () => onSelect(array.toArray(remove(keys)[0]));
  const { get, set } = Form.useContext();
  const channels = Form.useFieldValue(path).filter(({ key }) => keys.includes(key));
  const handleDuplicate = () => {
    const allChannels = get(path).value;
    onDuplicate?.(allChannels, keys);
  };
  const isEnabled = (c: C) => !c.disabled;
  const setEnabled = (key: string, enabled: boolean) =>
    set(deep.widen<Channel[]>(path)[key].disabled, !enabled);
  const handleDisable = () => keys.forEach((key) => setEnabled(key, false));
  const handleEnable = () => keys.forEach((key) => setEnabled(key, true));
  const canDuplicate = onDuplicate != null && keys.length > 0;
  const canRemove = keys.length > 0;
  const canDisable = channels.some(isEnabled);
  const canEnable = channels.some((c) => !isEnabled(c));
  return (
    <PlatformContextMenu.Menu>
      {!isPreview && (
        <>
          {canDuplicate && (
            <Menu.Item itemKey="duplicate" onClick={handleDuplicate}>
              <Icon.Copy />
              Duplicate
            </Menu.Item>
          )}
          <Menu.Divider />
          {contextMenuItems?.({ channels, keys }) ?? null}
          {canEnable && (
            <Menu.Item itemKey="enable" onClick={handleEnable}>
              <Icon.Enable />
              Enable
            </Menu.Item>
          )}
          {canDisable && (
            <Menu.Item itemKey="disable" onClick={handleDisable}>
              <Icon.Disable />
              Disable
            </Menu.Item>
          )}
        </>
      )}
      {tareable && (
        <TareItem channels={channels.filter((c) => !c.disabled).map(boundChannel)} />
      )}
      {!isPreview && (
        <>
          <Menu.Divider />
          {canRemove && <PlatformContextMenu.RemoveItem onClick={handleRemove} />}
          <Menu.Divider />
        </>
      )}
      <PlatformContextMenu.ReloadConsoleItem />
    </PlatformContextMenu.Menu>
  );
};

export interface ChannelListItemProps extends List.ItemProps<string> {
  /** The item shows a tare button for a float channel. */
  tareable?: boolean;
}

export interface ChannelListProps<C extends Channel>
  extends
    Omit<ContextMenuProps<C>, "keys">,
    Pick<Flex.BoxProps, "onDragOver" | "onDrop" | "grow" | "style"> {
  /**
   * Null for a list whose entries an outer BindChannels binds, or whose channels no
   * device maps.
   */
  resolve: BindChannelsProps<C>["resolve"] | null;
  emptyContent: ReactElement;
  header: ReactNode;
  /** Pinned below the list; a create action in the list's own idiom. */
  footer?: ReactNode;
  isDragging?: boolean;
  listItem: Component.RenderProp<ChannelListItemProps>;
  selected: string[];
  /**
   * Whether items select only through their checkbox or a modifier click. The selection
   * can then be empty.
   */
  checkable?: boolean;
  /** The columns items line up on. The list then shows their headings. */
  columns?: List.Column[];
  /**
   * Shown in place of the items, such as the channels by port. A right click on an
   * element with `Menu.CONTEXT_TARGET` and a channel key in `data-menu-key` opens that
   * channel's menu.
   */
  ports?: ReactNode;
}

export const ChannelList = <C extends Channel>({
  listItem,
  emptyContent,
  header,
  footer,
  isDragging,
  onDragOver,
  onDrop,
  selected,
  grow,
  style,
  resolve,
  checkable = false,
  columns,
  ports,
  ...rest
}: ChannelListProps<C>) => {
  const { onSelect, path, data, tareable = false } = rest;
  const { status } = Form.useFieldState(path);
  const handleChange = (keys: string[]) => onSelect(keys);
  const menuProps = Menu.useContextMenu();
  const headingsRef = useRef<HTMLDivElement>(null);
  const handleScroll = (e: UIEvent<HTMLDivElement>) => {
    if (headingsRef.current != null)
      headingsRef.current.scrollLeft = e.currentTarget.scrollLeft;
  };
  const content = (
    <Flex.Box className={CSS.B("channel-list")} empty grow={grow} style={style}>
      {resolve != null && <BindChannels<C> path={path} resolve={resolve} />}
      <Menu.ContextMenu
        {...menuProps}
        menu={(p) => <ContextMenu {...p} {...rest} />}
        onDragOver={onDragOver}
        onDrop={onDrop}
      >
        {/* A header element that names a channel by data-menu-key opens its menu. */}
        <Flex.Box
          y
          empty
          className={menuProps.className}
          onContextMenu={menuProps.open}
        >
          {header}
          {ports == null && columns != null && (
            <List.Headings ref={headingsRef} checkable={checkable} />
          )}
        </Flex.Box>
        {ports != null ? (
          <Flex.Box
            y
            grow
            empty
            className={menuProps.className}
            onContextMenu={menuProps.open}
          >
            {ports}
          </Flex.Box>
        ) : (
          <Select.Frame<string, C>
            multiple
            data={data}
            value={selected}
            onChange={handleChange}
            replaceOnSingle={!checkable}
            allowNone={checkable}
            autoSelectOnNone={!checkable}
          >
            <List.Scroll
              full="y"
              onDragOver={onDragOver}
              onDrop={onDrop}
              className={menuProps.className}
              onContextMenu={menuProps.open}
              onScroll={handleScroll}
            >
              <List.Items<string, C> emptyContent={emptyContent}>
                {tareable ? (p) => listItem({ ...p, tareable }) : listItem}
              </List.Items>
            </List.Scroll>
          </Select.Frame>
        )}
      </Menu.ContextMenu>
      {status.variant !== "success" && (
        <Input.HelpText
          variant={status.variant}
          className={CSS.BE("channel-list", "status")}
        >
          {status.message}
        </Input.HelpText>
      )}
      {footer}
    </Flex.Box>
  );
  if (columns == null) return content;
  return <List.Columns columns={columns}>{content}</List.Columns>;
};
