import { LineSVis } from "..";

import { TelemetryClient, TelemetryClientResponse } from "@/features/vis/telem/client";
import { AxisKey } from "@/features/vis/types";

export interface DataState {
  data: LineVisData;
  error: Error | null;
}

export type LineVisData = Record<AxisKey, TelemetryClientResponse[]>;

const ZERO_DATA: LineVisData = {
  y1: [],
  y2: [],
  y3: [],
  y4: [],
  x1: [],
  x2: [],
};

const initialDataState = (): DataState => ({
  data: { ...ZERO_DATA },
  error: null,
});

const fetchData = async (
  vis: LineSVis,
  client: TelemetryClient,
  isLive: boolean
): Promise<DataState> => {
  const keys = Object.values(vis.channels)
    .flat()
    .filter((key) => key.length > 0);
  const ranges = Object.values(vis.ranges).flat();
  let entries: TelemetryClientResponse[] = [];
  try {
    entries = await client.retrieve({
      keys,
      ranges,
      bypassCache: isLive,
    });
  } catch (error) {
    return { ...initialDataState(), error: error as Error };
  }
  const data: LineVisData = { ...ZERO_DATA };
  Object.values(vis.ranges).forEach((ranges) =>
    ranges.forEach((range) =>
      Object.entries(vis.channels).forEach(([axis, channelKeys]) => {
        if (!Array.isArray(channelKeys)) channelKeys = [channelKeys as string];
        data[axis as AxisKey] = data[axis as AxisKey].concat(
          entries.filter(
            ({ key, range: r }) => channelKeys.includes(key) && r === range
          )
        );
      })
    )
  );
  return { data, error: null };
};

export const Data = {
  fetch: fetchData,
  initial: initialDataState,
};
