import { ChannelKeys } from "@synnaxlabs/client";
import { TimeRange, TimeSpan, TimeRangeT, TimeSpanT } from "@synnaxlabs/x";

export interface Range {
  key: string;
  name: string;
  variant: "static" | "dynamic";
  timeRange: TimeRange;
  span: TimeSpan;
}

export const rangeFromPayload = (pld: RangePayload): Range => ({
  key: pld.key,
  name: pld.name,
  variant: pld.variant,
  timeRange: new TimeRange(pld.timeRange.start, pld.timeRange.end),
  span: new TimeSpan(pld.span),
});

export interface RangePayload {
  key: string;
  name: string;
  variant: "static" | "dynamic";
  timeRange: TimeRangeT;
  span: TimeSpanT;
}

export interface ChannelRange extends Range {
  channels: ChannelKeys;
}
