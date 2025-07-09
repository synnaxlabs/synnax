// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@/main.css";

import { type ranger, TimeSpan } from "@synnaxlabs/client";
import { type ReactElement, useEffect, useState } from "react";
import ReactDOM from "react-dom/client";

import { Align } from "@/align";
import { Component } from "@/component";
import { Input } from "@/input";
import { List } from "@/list";
import { Pluto } from "@/pluto";
import { Ranger } from "@/ranger";
import { Select } from "@/select";
import { Text } from "@/text";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  const { selected, hovered, onSelect } = Select.useItemState<ranger.Key>(itemKey);
  console.log(itemKey, selected, hovered);
  return (
    <List.Item
      itemKey={itemKey}
      {...rest}
      selected={selected}
      hovered={hovered}
      onSelect={onSelect}
    >
      <Text.Text level="p">{item?.name}</Text.Text>
    </List.Item>
  );
};

const listItem = Component.renderProp(ListItem);

const RangeList = () => {
  const { data, useListItem, retrieve } = Ranger.useList();
  const [searchTerm, setSearchTerm] = useState<string>("");
  const [selected, setSelected] = useState<string>("");
  const selectProps = Select.useSingle({
    data,
    onChange: setSelected,
    value: selected,
  });
  return (
    <Align.Space x>
      <Align.Space y>
        <Input.Text
          value={searchTerm}
          onChange={(v) => {
            setSearchTerm(v);
            retrieve(() => ({
              term: v,
              offset: 0,
              limit: 10,
            }));
          }}
        />
        {selected}
        {selectProps.hover}
      </Align.Space>
      <Select.Provider value={selected} {...selectProps}>
        <List.List data={data} useItem={useListItem}>
          <List.Items>{listItem}</List.Items>
        </List.List>
      </Select.Provider>
    </Align.Space>
  );
};

const Content = (): ReactElement => <RangeList />;

const Main = (): ReactElement => (
  <Pluto.Provider
    connParams={{
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
    }}
  >
    <Content />
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
