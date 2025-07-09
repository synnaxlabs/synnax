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

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement, useState } from "react";
import ReactDOM from "react-dom/client";

import { Align } from "@/align";
import { Component } from "@/component";
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
  const [selected, setSelected] = useState<string>("");
  return (
    <Align.Space y style={{ padding: "10rem" }}>
      <Ranger.SelectSingle value={selected} onChange={setSelected} />
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
