// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { List } from ".";

import { componentRenderProp } from "@/util/renderProp";

const story: Meta<typeof List> = {
  title: "Core/Standard/List",
  component: List,
};

const dataTypes = [
  "float64",
  "int64",
  "string",
  "bool",
  "date",
  "time",
  "datetime",
  "duration",
  "bytes",
  "struct",
  "list",
  "map",
  "null",
];

const namePrefixes = ["strainGauge", "accelerometer", "gyroscope", "magnetometer"];

const data = Array.from({ length: 500 }, (_, i) => ({
  key: `key-${i}`,
  dataType: dataTypes[Math.floor(Math.random() * dataTypes.length)],
  name: `${namePrefixes[Math.floor(Math.random() * namePrefixes.length)]}-${i}`,
  rate: Math.floor(Math.random() * 100),
}));

const columns = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "dataType",
    name: "Data Type",
  },
  {
    key: "rate",
    name: "Rate",
  },
];

export const Column: StoryFn<typeof List> = () => (
  <List data={data}>
    <List.Column.Header columns={columns} />
    <List.Core.Virtual itemHeight={List.Column.itemHeight} style={{ height: "80vh" }}>
      {List.Column.Item}
    </List.Core.Virtual>
  </List>
);

export const Search: StoryFn<typeof List> = () => (
  <List data={data}>
    <List.Search />
    <List.Column.Header columns={columns} />
    <List.Core.Virtual itemHeight={List.Column.itemHeight} style={{ height: "80vh" }}>
      {componentRenderProp(List.Column.Item)}
    </List.Core.Virtual>
  </List>
);

export const Selectable: StoryFn<typeof List> = () => {
  const [selected, setSelected] = useState<readonly string[]>([]);
  return (
    <List data={data}>
      <List.Selector value={selected} onChange={setSelected} />
      <List.Column.Header columns={columns} />
      <List.Core.Virtual itemHeight={List.Column.itemHeight} style={{ height: "80vh" }}>
        {List.Column.Item}
      </List.Core.Virtual>
    </List>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
