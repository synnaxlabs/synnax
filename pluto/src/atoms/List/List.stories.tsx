import { useState } from "react";

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { List } from ".";

const story: ComponentMeta<typeof List> = {
  title: "Atoms/List",
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

const data = Array.from({ length: 400 }, (_, i) => ({
  key: `key-${i}`,
  dataType: dataTypes[Math.floor(Math.random() * dataTypes.length)],
  name: `${namePrefixes[Math.floor(Math.random() * namePrefixes.length)]}-${i}`,
  rate: Math.floor(Math.random() * 100),
}));

const columns = [
  {
    key: "name",
    label: "Name",
  },
  {
    key: "dataType",
    label: "Data Type",
  },
  {
    key: "rate",
    label: "Rate",
  },
];

export const Column: ComponentStory<typeof List> = () => (
  <List data={data}>
    <List.Column.Header columns={columns} />
    <List.Core.Virtual itemHeight={List.Column.itemHeight} style={{ height: "80vh" }}>
      {List.Column.Item}
    </List.Core.Virtual>
  </List>
);

export const Search: ComponentStory<typeof List> = () => (
  <List data={data}>
    <List.Search />
    <List.Column.Header columns={columns} />
    <List.Core.Virtual itemHeight={List.Column.itemHeight} style={{ height: "80vh" }}>
      {List.Column.Item}
    </List.Core.Virtual>
  </List>
);

export const Selectable: ComponentStory<typeof List> = () => {
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

export default story;
