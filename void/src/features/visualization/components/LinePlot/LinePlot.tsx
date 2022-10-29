import { Synnax, ChannelPayload } from "@synnaxlabs/client";
import {
  AutoSize,
  LinePlot as PlutoLinePlot,
  PlotData,
  Select,
  Space,
  Theming,
} from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";
import { LinePlotVisualization, Visualization } from "../../types";
import { LinePlotControls } from "./LinePlotControls";
import "./LinePlot.css";

export interface LinePlotProps {
  visualization: LinePlotVisualization;
  onChange: (vis: Visualization) => void;
  client: Synnax;
  resizeDebounce: number;
}

export const LinePlot = ({
  visualization,
  client,
  onChange,
  resizeDebounce,
}: LinePlotProps) => {
  const { axes, series, channels } = visualization;
  const [data, setData] = useState<PlotData>({});
  const {
    theme: { colors },
  } = Theming.useContext();

  const [channelOpts, setChannelOpts] = useState<ChannelPayload>([]);

  useEffect(() => {
    const fn = async () => {
      const channels = await client.channel.retrieveAll();
      console.log(channels);
      setChannelOpts(channels.map((ch) => ch.payload));
    };
    fn();
  }, [client]);

  const onChannelSelect = (channels: string[]) => {
    const fn = async () => {
      const nextData: PlotData = {};
      for (const channel of channels) {
        const data = await client.data.read(channel, 0, 900000000000000);
        nextData[channel] = data;
        // if channel is last channel
        if (channels.indexOf(channel) === channels.length - 1) {
          nextData["time"] = Array.from({ length: data.length }, (_, i) => i);
        }
      }
      setData(nextData);
      onChange({
        ...visualization,
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
      } as Visualization);
    };
    fn();
  };

  useEffect(() => {
    onChannelSelect(channels);
  }, []);

  return (
    <div className="void-line-plot__container">
      <AutoSize
        className="void-line-plot__plot__container"
        debounce={resizeDebounce}
      >
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
      <Space direction="vertical">
        <Select.Multiple
          selected={channels}
          onSelect={onChannelSelect}
          options={channelOpts}
          tagKey="name"
          listPosition="top"
          columns={[
            {
              key: "name",
              label: "Name",
            },
          ]}
        />
      </Space>
    </div>
  );
};
