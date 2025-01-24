// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Form, List, Menu as PMenu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { CSS } from "@/css";

export interface Channel {
  key: string;
  enabled: boolean;
}

export interface ChannelListItemProps<C extends Channel>
  extends List.ItemProps<string, C> {
  path: string;
  isSnapshot: boolean;
}

export type ChannelListProps<C extends Channel> = Omit<
  Align.SpaceProps,
  "children" | "onSelect"
> & {
  children: (props: ChannelListItemProps<C>) => ReactElement;
  header: ReactElement;
  isSnapshot: boolean;
  emptyContent: ReactElement;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
  channels: C[];
  onTare?: (keys: string[], channels: C[]) => void;
  allowTare?: (keys: string[], channels: C[]) => boolean;
  path: string;
  remove: (index: number | number[]) => void;
};

export const ChannelList = <C extends Channel>({
  children,
  header,
  isSnapshot,
  emptyContent,
  onSelect,
  onTare,
  allowTare,
  selected,
  channels,
  path,
  remove,
  ...props
}: ChannelListProps<C>): ReactElement => {
  const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement | null => {
    const keyToIndexMap = new Map(channels.map(({ key }, i) => [key, i]));
    const indices = keys.map((key) => keyToIndexMap.get(key)).filter((i) => i != null);
    const handleRemove = () => {
      remove(indices);
      onSelect([], -1);
    };
    const { set } = Form.useContext();
    const handleDisable = () =>
      indices.forEach((index) => set(`${path}.${index}.enabled`, false));
    const handleEnable = () =>
      indices.forEach((index) => set(`${path}.${index}.enabled`, true));
    const handleSelect: Record<string, () => void> = {
      remove: handleRemove,
      disable: handleDisable,
      enable: handleEnable,
      tare: () => onTare?.(keys, channels),
    };
    const canDisable = indices.some((i) => channels[i].enabled);
    const canEnable = indices.some((i) => !channels[i].enabled);
    const canTare = allowTare?.(keys, channels);
    return (
      <PMenu.Menu onChange={handleSelect} level="small">
        {!isSnapshot && (
          <>
            <PMenu.Item itemKey="remove" startIcon={<Icon.Close />}>
              Remove
            </PMenu.Item>
            <PMenu.Divider />
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
            {(canEnable || canDisable) && <PMenu.Divider />}
            {canTare === true && (
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
  const menuProps = PMenu.useContextMenu();
  return (
    <Align.Space grow className={CSS.B("channels")} {...props} empty>
      {header}
      <PMenu.ContextMenu {...menuProps} menu={(p) => <ContextMenu {...p} />}>
        <List.List<string, C> data={channels} emptyContent={emptyContent}>
          <List.Selector<string, C>
            value={selected}
            replaceOnSingle
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
          >
            <List.Core<string, C> grow>
              {(props) =>
                children({ ...props, path: `${path}.${props.index}`, isSnapshot })
              }
            </List.Core>
          </List.Selector>
        </List.List>
      </PMenu.ContextMenu>
    </Align.Space>
  );
};
