import { AutoSize, Space, LinePlot, Select } from "@synnaxlabs/pluto";

const data = {
  a: Array.from({ length: 1000 }, (_, i) => i),
  b: Array.from(
    { length: 1000 },
    (_, i) => Math.sin(i / 10) * 2 + Math.random() * 0.1
  ),
  c: Array.from(
    { length: 1000 },
    (_, i) => Math.cos(i / 20) + Math.random() * 0.1
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

const options = Array.from({ length: 500 }, (_, i) => ({
  key: i,
  name: `Option ${i}`,
}));

export default function Plot() {
  return (
    <Space style={{ width: "100%", height: "100%", padding: "12px" }}>
      <AutoSize style={{ flexGrow: 1, overflow: "hidden" }}>
        {({ width, height }) => (
          <LinePlot
            width={width}
            height={height}
            data={data}
            axes={axes}
            series={series}
          />
        )}
      </AutoSize>
      <Select.Multiple
        listPosition="top"
        tagKey="name"
        columns={[
          {
            key: "name",
            label: "Name",
            visible: true,
          },
        ]}
        options={options}
      />
    </Space>
  );
}
