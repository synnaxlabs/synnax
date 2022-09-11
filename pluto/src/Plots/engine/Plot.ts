import { DataProvider } from "./Data";

type Channel = {
  key: string;
  color: string;
  visible: boolean;
  name?: string;
  timeRange?: [number, number];
};

type PlotProps = {
  bounds: HTMLElement;
  resize: boolean;
};

type PlotState = {
  channels: Channel[];
};

export default class Plot {
  bounds: HTMLElement;
  state: PlotState;

  constructor(props: PlotProps) {
    this.bounds = props.bounds;
    this.state = {
      channels: [],
    };
  }

  addChannel(channel: Channel) {
    this.state.channels.push(channel);
  }

  removeChannel(key: string) {
    this.state.channels = this.state.channels.filter(
      (channel) => channel.key !== key
    );
  }
}
