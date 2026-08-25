// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import {
  Access,
  Flex,
  Input,
  List,
  Select,
  stopPropagation,
  Text,
} from "@synnaxlabs/pluto";

export interface ItemProps extends List.ItemProps<arc.Key> {
  onRename?: (name: string) => void;
  textIdPrefix?: string;
}

export const Item = ({ onRename, textIdPrefix = "text", ...props }: ItemProps) => {
  const { itemKey } = props;
  const item = List.useItem<arc.Key, arc.Arc>(itemKey);
  const { onSelect, selected, hovered } = Select.useItemState(itemKey);
  const canRename = Access.useUpdateGranted(arc.ontologyID(itemKey));

  if (item == null) return null;
  const { name } = item;

  return (
    <List.Item
      {...props}
      hovered={hovered}
      selected={selected}
      rounded={!selected}
      onSelect={onSelect}
      justify="between"
      align="center"
    >
      <Flex.Box x align="center">
        <Input.Checkbox
          value={selected}
          onChange={onSelect}
          onClick={stopPropagation}
          aria-label="Select"
        />
        <Text.MaybeEditable
          id={`${textIdPrefix}-${itemKey}`}
          level="p"
          value={name}
          onChange={canRename ? onRename : undefined}
          allowDoubleClick={false}
        />
      </Flex.Box>
    </List.Item>
  );
};
