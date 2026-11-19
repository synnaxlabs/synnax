// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { Button } from "@synnaxlabs/lyra/button";
import { type Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type Schematic } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";

interface StateListProps extends Input.Control<number> {
  tabs: Schematic.Node.Custom.Value[];
}

interface StateListItemProps extends List.ItemRenderProps<number> {
  name: string;
}

const StateListItem = ({ itemKey, name }: StateListItemProps) => {
  const { selected, onSelect } = Select.useItemState(itemKey);
  return (
    <Button.Button
      variant={selected ? "filled" : "outlined"}
      justify="center"
      onClick={onSelect}
      className={CSS.B("schematic-state-list-item")}
      grow
    >
      {name}
    </Button.Button>
  );
};

export const StateList = ({ value, onChange, tabs }: StateListProps) => {
  if (tabs.length === 1) return null;
  return (
    <Select.Frame
      value={value}
      onChange={onChange}
      data={tabs.map((t) => t.value)}
      closeDialogOnSelect={false}
    >
      <List.Scroll x gap={1}>
        <List.Items<number>>
          {({ key, ...rest }) => (
            <StateListItem
              key={key}
              {...rest}
              name={tabs.find((t) => t.value === rest.itemKey)?.name ?? ""}
            />
          )}
        </List.Items>
      </List.Scroll>
    </Select.Frame>
  );
};
