// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Form, List, Menu as PMenu, type RenderProp } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { type Channel } from "@/hardware/common/task/types";

interface ContextMenuProps<C extends Channel> {
  allowTare?: (keys: string[], channels: C[]) => boolean;
  channels: C[];
  isSnapshot: boolean;
  keys: string[];
  onDuplicate?: (channels: C[], indices: number[]) => void;
  onSelect: (keys: string[], index: number) => void;
  onTare?: (keys: string[], channels: C[]) => void;
  path: string;
  remove: (index: number | number[]) => void;
}

const ContextMenu = <C extends Channel>({
  allowTare,
  channels,
  isSnapshot,
  keys,
  onDuplicate,
  onSelect,
  onTare,
  path,
  remove,
}: ContextMenuProps<C>) => {
  const keyToIndexMap = new Map(channels.map(({ key }, i) => [key, i]));
  const indices = keys.map((key) => keyToIndexMap.get(key)).filter((i) => i != null);
  const handleRemove = () => {
    if (indices.length === 0) return onSelect([], -1);
    remove(indices);
    const sorted = indices.sort((a, b) => a - b);
    const idxToSelect = sorted[0] - 1;
    if (idxToSelect >= 0) onSelect([channels[idxToSelect].key], idxToSelect);
    else onSelect([], -1);
  };
  const { set } = Form.useContext();
  const handleDuplicate = () => onDuplicate?.(channels, indices);
  const handleDisable = () =>
    indices.forEach((index) => set(`${path}.${index}.enabled`, false));
  const handleEnable = () =>
    indices.forEach((index) => set(`${path}.${index}.enabled`, true));
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
  const canDuplicate = onDuplicate != null && indices.length > 0;
  const canRemove = indices.length > 0;
  const canDisable = indices.some((i) => channels[i].enabled);
  const canEnable = indices.some((i) => !channels[i].enabled);
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
          {canDuplicate || (canRemove && <PMenu.Divider />)}
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

export interface ChannelListItemProps<C extends Channel>
  extends List.ItemProps<string, C> {
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
  listItem: RenderProp<ChannelListItemProps<C>>;
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
  const { channels, isSnapshot, onSelect, path } = rest;
  const handleChange = useCallback(
    (keys: string[], { clickedIndex }: { clickedIndex: number | null }) =>
      clickedIndex != null && onSelect(keys, clickedIndex),
    [onSelect],
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
        style={{ height: "calc(100% - 5.5rem)" }}
      >
        <List.List<string, C> data={channels} emptyContent={emptyContent}>
          <List.Selector<string, C>
            onChange={handleChange}
            replaceOnSingle
            value={selected}
          >
            <List.Core<string, C>>
              {(props) =>
                listItem({ isSnapshot, path: `${path}.${props.index}`, ...props })
              }
            </List.Core>
          </List.Selector>
        </List.List>
      </PMenu.ContextMenu>
    </Align.Space>
  );
};
