import { useEffect, useRef, useState } from "react";

import { Synnax } from "@synnaxlabs/client";
import { AutoSize, LinePlot as PlutoLinePlot, Theming } from "@synnaxlabs/pluto";
import type { PlotData } from "@synnaxlabs/pluto";

import {
  LinePlotVisualization,
  SugaredLinePlotVisualization,
  Visualization,
} from "../../types";

import { LinePlotControls } from "./LinePlotControls";
import "./LinePlot.css";

export interface LinePlotProps {
  visualization: SugaredLinePlotVisualization;
  onChange: (vis: Visualization) => void;
  client: Synnax;
  resizeDebounce: number;
}

function usePrevious<V>(value: V): V | undefined {
  const ref = useRef<V>();
  useEffect(() => {
    ref.current = value;
  }, [value]);
  return ref.current;
}

export const LinePlot = ({
  visualization,
  client,
  onChange,
  resizeDebounce,
}: LinePlotProps): JSX.Element => {
  const { axes, series, channels, ranges } = visualization;
  const [data, setData] = useState<PlotData>({});
  const {
    theme: { colors },
  } = Theming.useContext();
  const prevVisu = usePrevious(visualization);

  useEffect(() => {
    if (
      prevVisu != null &&
      prevVisu.channels.length === visualization.channels.length &&
      prevVisu.ranges.length === visualization.ranges.length
    )
      return;
    void (async () => {
      const nextData: PlotData = {};
      for (const range of ranges) {
        for (const key of channels) {
          const data = await client.data.read(key, range.start, range.end);
          if (data != null) nextData[key] = data as unknown as any[];
          if (channels.indexOf(key) === channels.length - 1) {
            nextData.time = Array.from({ length: data?.length ?? 0 }, (_, i) => i);
          }
        }
      }
      setData(nextData);
      const nextV: LinePlotVisualization = {
        ...visualization,
        ranges: ranges.map((range) => range.key),
        series: channels.map((ch) => ({
          label: ch,
          x: "time",
          y: ch,
          color: colors.visualization.palettes.default[channels.indexOf(ch)],
          axis: "y",
        })),
        axes: [
          {
            key: "x",
            location: "bottom",
            label: "x",
          },
          {
            key: "y",
            location: "left",
            label: "y",
          },
        ],
        channels,
      };
      onChange(nextV);
    })();
  }, [client, channels, ranges]);

  return (
    <div className="delta-line-plot__container">
      <AutoSize className="delta-line-plot__plot__container" debounce={resizeDebounce}>
        {({ width, height }) => (
          <PlutoLinePlot
            width={width}
            height={height}
            data={data}
            axes={axes}
            series={series}
          />
        )}
      </AutoSize>
      <LinePlotControls
        visualization={visualization}
        onChange={onChange}
        client={client}
      />
    </div>
  );
};
