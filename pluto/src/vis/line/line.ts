import { Channels } from "./channels";
import { Ranges } from "./ranges";

export class LinePlot {
  channels: Channels;
  ranges: Ranges;

  constructor() {
    this.channels = Channels.zero();
    this.ranges = Ranges.zero();
  }
}
