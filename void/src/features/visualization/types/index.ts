import { LinePlotMetadata } from "@synnaxlabs/pluto";

export type Visualization = {
  layoutKey: string;
  channels: string[];
} & LinePlotMetadata;
