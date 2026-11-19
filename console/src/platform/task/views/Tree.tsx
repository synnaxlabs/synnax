// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/views/Tree.css";

import { type channel } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { type Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Channel } from "@synnaxlabs/pluto";
import { type deep, primitive, type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import {
  isMissing,
  Message,
  MISSING_DESCRIPTIONS,
  NEW_DESCRIPTION,
  STATE_MESSAGES,
  StateIcon,
} from "@/platform/task/ChannelState";
import { useIsPreview } from "@/platform/task/useIsPreview";

export interface ChannelIconProps {
  /** The form path of the channel, which is 0 until deploying creates it. */
  channel: deep.Target<channel.Key>;
  /** Deploying the task skips the channel. */
  skipped: boolean;
}

/** Marks a tree item whose channel deploying creates, or whose channel is gone. */
export const ChannelIcon = ({
  channel,
  skipped,
}: ChannelIconProps): ReactElement | null => {
  const key = Form.useFieldValue(channel);
  const result = Channel.useResultName(primitive.isNonZero(key) ? { key } : null);
  if (!primitive.isNonZero(key))
    return skipped ? null : (
      <Message
        variant="info"
        message={STATE_MESSAGES.new}
        description={NEW_DESCRIPTION}
      >
        <StateIcon state="new" />
      </Message>
    );
  if (!isMissing(result)) return null;
  return (
    <Message
      variant="warning"
      message={STATE_MESSAGES.missing}
      description={MISSING_DESCRIPTIONS[skipped ? "skips" : "creates"]}
    >
      <StateIcon state="missing" />
    </Message>
  );
};

export interface TreeGroupItemProps extends Tree.ItemRenderProps<string> {
  /** Names the group. */
  label: ReactNode;
  /**
   * The form path of the channel of a leaf: a group with no fields, read whole into
   * one channel.
   */
  channel?: deep.Target<channel.Key>;
  /** Shows the group as switched off. */
  off?: boolean;
  /** Trails the label, such as an add button and an enabled checkbox. */
  children?: ReactNode;
}

/** A group of a task tree, such as an endpoint, with its fields beneath. */
export const TreeGroupItem = ({
  label,
  channel,
  off = false,
  children,
  ...props
}: TreeGroupItemProps): ReactElement => (
  <Tree.Item
    {...props}
    className={CSS.cls(
      CSS.B("group-item"),
      channel != null && CSS.M("leaf"),
      off && CSS.M("off"),
    )}
  >
    {label}
    {channel != null && <ChannelIcon channel={channel} skipped={off} />}
    {children}
  </Tree.Item>
);

export interface TreeFieldItemProps extends Tree.ItemRenderProps<string> {
  /** Names the field. */
  label: string;
  /** Stands in, in gray, for an empty label. */
  emptyLabel: string;
  /** The form path of the field's channel, or null when the field has none. */
  channel: deep.Target<channel.Key> | null;
  /** Shows the field as switched off: its label is struck through. */
  off: boolean;
  /** Deploying the task skips the channel. */
  skipped: boolean;
  /** Trails the label, such as an enabled checkbox. */
  children?: ReactNode;
}

/** A field of a task tree, read into one channel. */
export const TreeFieldItem = ({
  label,
  emptyLabel,
  channel,
  off,
  skipped,
  children,
  ...props
}: TreeFieldItemProps): ReactElement => (
  <Tree.Item {...props} className={CSS.cls(CSS.B("field-item"), off && CSS.M("off"))}>
    <Text.Text
      level="small"
      weight={500}
      color={label === "" ? 8 : 10}
      overflow="fade"
      className={CSS.B("field-pointer")}
    >
      {label === "" ? emptyLabel : label}
    </Text.Text>
    {channel != null && <ChannelIcon channel={channel} skipped={skipped} />}
    {children}
  </Tree.Item>
);

export interface TreeListProps extends Tree.UseReturn<string> {
  onSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"];
  /** The menu of a right click on the selected items. */
  menu: Component.RenderProp<Menu.ContextMenuMenuProps>;
  /** Fills the tree while it holds nothing: "No endpoints". */
  emptyMessage: string;
  /** Labels the add button under the tree: "Add endpoint". */
  addLabel: string;
  onAdd: () => void;
  addDisabled?: boolean;
  children: Component.RenderProp<Tree.ItemRenderProps<string>>;
}

const ITEM_HEIGHT = 36;

/**
 * The list column of a task tree: the tree, its right-click menu, and an add button
 * under it outside preview.
 */
export const TreeList = ({
  menu,
  emptyMessage,
  addLabel,
  onAdd,
  addDisabled = false,
  children,
  ...props
}: TreeListProps): ReactElement => {
  const isPreview = useIsPreview();
  const menuProps = Menu.useContextMenu();
  return (
    <>
      <Menu.ContextMenu {...menuProps} menu={menu}>
        <Tree.Tree<string, record.Keyed<string>>
          {...props}
          itemHeight={ITEM_HEIGHT}
          className={CSS.cls(menuProps.className, CSS.B("task-tree"))}
          onContextMenu={menuProps.open}
          emptyContent={<Empty.Action message={emptyMessage} />}
          allowNone={false}
          autoSelectOnNone
        >
          {children}
        </Tree.Tree>
      </Menu.ContextMenu>
      {!isPreview && (
        <Button.CreateListItem
          size="small"
          onClick={onAdd}
          disabled={addDisabled}
          className={CSS.BE("panes", "create")}
        >
          {addLabel}
        </Button.CreateListItem>
      )}
    </>
  );
};
