// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Form, List, Menu } from "@synnaxlabs/pluto";
import { type Keyed } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { ChannelListContextMenu } from "@/hardware/common/task/ChannelListContextMenu";

interface BaseChannel extends Keyed<string> {
  enabled: boolean;
}

export interface ChannelListItemProps<C extends BaseChannel>
  extends List.ItemProps<string, C> {
  path: string;
  snapshot?: boolean;
}

interface Push<C extends BaseChannel> {
  push: (value: C | C[]) => void;
}

interface OnDuplicate<C extends BaseChannel> extends Push<C> {
  value: C[];
}

interface OnDupplicateFn {
  (indices: number[]): void;
}

type OnDuplicateCreator<C extends BaseChannel> = (
  props: OnDuplicate<C>,
) => OnDupplicateFn;

export interface ChannelListProps<C extends BaseChannel>
  extends Omit<Align.SpaceProps, "children" | "onSelect"> {
  path: string;
  snapshot?: boolean;
  children: (props: ChannelListItemProps<C>) => ReactElement;
  header: (props: Push<C>) => ReactElement;
  emptyContent: (props: Push<C>) => ReactElement;
  selected: string[];
  onSelect: (keys: string[], index: number) => void;
  allowTare?: (value: C[]) => boolean;
  onTare?: (keys: number[]) => void;
  onDuplicate?: OnDuplicateCreator<C>;
}

export const ChannelList = <C extends BaseChannel>({
  children,
  path,
  header,
  snapshot,
  emptyContent,
  onSelect,
  selected,
  allowTare,
  onTare,
  onDuplicate,
}: ChannelListProps<C>): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<C>({ path });
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space>
      {header({ push })}
      <Menu.ContextMenu
        {...menuProps}
        menu={({ keys }: Menu.ContextMenuMenuProps) => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            snapshot={snapshot}
            allowTare={allowTare?.(value)}
            onTare={onTare}
            onDuplicate={onDuplicate?.({ push, value })}
          />
        )}
      >
        <List.List<string, C> data={value} emptyContent={emptyContent({ push })}>
          <List.Selector<string, C>
            value={selected}
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
          >
            <List.Core<string, C>>
              {(props) =>
                children({ ...props, path: `${path}.${props.index}`, snapshot })
              }
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
    </Align.Space>
  );
};
