import {
  AutoSizer,
  Space,
  Plot as PlutoPlot,
  MultiSelect,
} from "@synnaxlabs/pluto";

const data = {
  a: Array.from({ length: 100000 }, (_, i) => i),
  b: Array.from(
    { length: 100000 },
    (_, i) => Math.sin(i / 1000) * 2 + Math.random() * 0.1
  ),
  c: Array.from(
    { length: 100000 },
    (_, i) => Math.cos(i / 1900) + Math.random() * 0.1
  ),
};

const axes = [
  {
    key: "x",
    label: "X",
    location: "bottom",
  },
  {
    key: "y",
    label: "Y",
    location: "left",
  },
  {
    key: "y2",
    label: "Y2",
    location: "right",
  },
];

const series = [
  {
    label: "Series 1",
    x: "a",
    y: "b",
    axis: "y",
  },
  {
    label: "Series 1",
    x: "a",
    y: "c",
    axis: "y2",
  },
];

export default function Plot() {
  return (
    <Space style={{ width: "100%", height: "100%", padding: "12px" }}>
      <AutoSizer style={{ flexGrow: 1, overflow: "hidden" }} debounce={100}>
        {({ width, height }) => (
          <PlutoPlot
            width={width}
            height={height}
            data={data}
            axes={axes}
            series={series}
          />
        )}
      </AutoSizer>
      <MultiSelect
        listPosition="top"
        columns={[
          {
            key: "name",
            label: "Name",
            visible: true,
          },
        ]}
        options={[
          {
            name: "Series 1",
          },
        ]}
      />
    </Space>
  );
}
