import { useClusterClient } from "@/features/cluster";
import { LayoutRendererProps } from "@/features/layout";
import { AutoSize, LinePlot, PlotData, Axis } from "@synnaxlabs/pluto";
import { memo, useEffect, useState } from "react";
import { useSelectVisualization } from "../../store";
import { Visualization } from "../../types";

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

export const Plot = memo(({ layoutKey }: LayoutRendererProps) => {
  const vis = useSelectVisualization(layoutKey) as Visualization;
  if (!vis) return null;
  const client = useClusterClient();
  const [data, setData] = useState<PlotData>({ a: [], b: [] });

  useEffect(() => {
    if (!client) return;

    const fn = async () => {
      const ch = (await client.channel.retrieveByKeys(...vis.channels))[0];
      const chData = await ch.read(0, 9000000000000000000);
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
            axes={axes as Axis[]}
            series={series}
          />
        )}
      </AutoSize>
    </div>
  );
});
