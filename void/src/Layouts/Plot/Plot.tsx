import { TimeStamp } from "@synnaxlabs/client";
import { AutoSize, Space, LinePlot, Select, Data } from "@synnaxlabs/pluto";
import { memo, useEffect, useState } from "react";
import { useActiveClient } from "../../features/cluster/components/useActiveClient";

// const data = {
//   a: Array.from({ length: 1000 }, (_, i) => i),
//   b: Array.from(
//     { length: 1000 },
//     (_, i) => Math.sin(i / 10) * 2 + Math.random() * 0.1
//   ),
//   c: Array.from(
//     { length: 1000 },
//     (_, i) => Math.cos(i / 20) + Math.random() * 0.1
//   ),
// };

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
];

const series = [
  {
    label: "Series 1",
    x: "b",
    y: "a",
    axis: "y",
  },
];

const options = Array.from({ length: 500 }, (_, i) => ({
  key: i,
  name: `Option ${i}`,
}));

const Plot = () => {
  const client = useActiveClient();
  const [data, setData] = useState<Data>({ a: [], b: [] });

  useEffect(() => {
    if (!client) return;

    const fn = async () => {
      const ch = (await client.channel.retrieveByKeys("1-1"))[0];
      console.log(ch.key);
      const chData = await ch.read(0, 9000000000000000000);
      console.log(chData);
      let _data = {
        b: Array.from({ length: chData.length }, (_, i) => i),
        a: chData,
      };
      setData(_data);
    };
    fn();
  }, [client]);

  return (
    <div
      style={{
        overflow: "hidden",
        height: "100%",
        width: "100%",
        padding: "2rem",
      }}
    >
      <AutoSize
        style={{ height: "calc(100% - 36px)", overflow: "hidden" }}
        debounce={100}
      >
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
    </div>
  );
};

export default memo(Plot);
