// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer, status as cstatus } from "@synnaxlabs/client";
import {
  type Aggregation,
  aggregationZ,
  bounds,
  type breaker,
  DataType,
  MultiSeries,
  primitive,
  resolveWindow,
  type Series,
  sync,
  TimeRange,
  TimeStamp,
  windowZ,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  AbstractSource,
  LOADING_STATUS,
  type NumberSourceSpec,
  type Result,
  type SeriesSource,
  type SeriesSourceSpec,
  type Source,
  type Spec,
  type StringSourceSpec,
  success,
  SUCCESS_STATUS,
  type Telem,
  type ValueProps,
} from "@/telem/aether/telem";

/** The slice of a Synnax client that remote telemetry sources consume. */
export interface Client {
  feed: Pick<framer.Feed, "subscribe">;
  channels: {
    retrieve: (ch: channel.Key | channel.Name) => Promise<channel.Channel>;
  };
  /** @returns the current time on the Core's clock, which stamps every sample. */
  now: () => TimeStamp;
}

/** The status of a remote source created while the Core is disconnected. */
export const DISCONNECTED_STATUS: cstatus.Status = {
  key: "disconnected",
  name: "",
  variant: "warning",
  message: "Core disconnected",
  description: "",
  time: TimeStamp.ZERO,
};

const isProblem = (s?: cstatus.Status): boolean =>
  s?.variant === "warning" || s?.variant === "error";

// Whether next starts, changes, or ends a warning or an error. Other changes, such as
// the end of a load, give listeners nothing to show.
const problemChanged = (
  prev: cstatus.Status | undefined,
  next: cstatus.Status,
): boolean =>
  (isProblem(prev) || isProblem(next)) &&
  (prev?.variant !== next.variant ||
    prev.message !== next.message ||
    prev.description !== next.description);

// Holds a source's last failure. Retries repeat, so onChange runs only when the failure
// changes.
class FailureReporter {
  status?: cstatus.Status;
  constructor(private readonly onChange: () => void) {}

  report(e: unknown, message: string): void {
    const prev = this.status;
    const next = cstatus.fromException(e, message);
    this.status = next;
    if (problemChanged(prev, next)) this.onChange();
  }

  reset(): void {
    this.status = undefined;
  }
}

// The status of a source before its channel resolves, or after its retrieval fails.
const sourceStatus = (
  client: Client | null,
  loading: boolean,
  failures: FailureReporter,
): cstatus.Status => {
  if (client == null) return DISCONNECTED_STATUS;
  if (loading) return LOADING_STATUS;
  return failures.status ?? SUCCESS_STATUS;
};

export const streamChannelValuePropsZ = z.object({
  channel: z.number().or(z.string()),
});

export type StreamChannelValueProps = z.infer<typeof streamChannelValuePropsZ>;

/** Converts the newest sample of a non-empty series to a source's value. */
interface Conversion<V> {
  /** The value of a channel with no samples. */
  empty: V;
  convert: (series: Series) => V;
}

// A boolean reads as 0 or 1, and an int64 as its nearest number.
const NUMBER: Conversion<number> = {
  empty: NaN,
  convert: (series) => Number(series.at(-1, true)),
};

// Keeps variable density data types as text rather than coercing them to a number.
const STRING: Conversion<string> = {
  empty: "",
  convert: (series) => series.asString(-1, true),
};

interface Converted<V> {
  series: Series;
  length: number;
  value: V;
}

/** A source of the newest sample of a channel, converted to V. */
class ChannelValue<V>
  extends AbstractSource<typeof streamChannelValuePropsZ>
  implements Source<V>
{
  static readonly NUMBER_TYPE = "stream-channel-value";
  static readonly STRING_TYPE = "stream-channel-string-value";
  schema = streamChannelValuePropsZ;

  private readonly client: Client | null;
  private readonly conversion: Conversion<V>;
  private readonly failures = new FailureReporter(() => {
    this.notifiedStatus = this.failures.status;
    this.notify();
  });
  private subscription: framer.Subscription | null = null;
  private key: channel.Key | null = null;
  private converted: Converted<V> | null = null;
  // notified and notifiedStatus are the value and status listeners last heard of. A
  // read can convert a new sample before the subscription's notice runs.
  private notified: Converted<V> | null = null;
  private notifiedStatus?: cstatus.Status;
  private generation = 0;
  private valid = false;
  private loading = true;

  constructor(client: Client | null, props: unknown, conversion: Conversion<V>) {
    super(props);
    this.client = client;
    this.conversion = conversion;
  }

  value(): Result<V> {
    const { empty } = this.conversion;
    if (primitive.isZero(this.props.channel)) return success(empty);
    if (!this.valid) void this.resolve();
    const { subscription, key } = this;
    if (subscription == null || key == null)
      return {
        data: empty,
        status: sourceStatus(this.client, this.loading, this.failures),
        appended: 0,
      };
    const snapshot = subscription.value(key);
    this.convert(snapshot);
    const { status, time, appended } = snapshot;
    return { data: this.converted?.value ?? empty, status, time, appended };
  }

  cleanup(): void {
    this.generation++;
    this.subscription?.close();
    this.subscription = null;
    this.key = null;
    this.converted = null;
    this.notified = null;
    this.notifiedStatus = undefined;
    this.valid = false;
    this.loading = true;
    this.failures.reset();
  }

  // Converts the newest sample when its series or length changes, since a string
  // conversion scans the series. A successful read with no samples, as after a delete,
  // clears it.
  private convert({ series, status }: framer.Snapshot): void {
    const last = series.series.at(-1);
    if (last == null || last.length === 0) {
      if (status.variant === "success") this.converted = null;
      return;
    }
    const prev = this.converted;
    if (prev?.series === last && prev.length === last.length) return;
    this.converted = {
      series: last,
      length: last.length,
      value: this.conversion.convert(last),
    };
  }

  /** Never rejects: a failure invalidates the source and sets its status. */
  private async resolve(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) return;
    try {
      const { key } = await client.channels.retrieve(this.props.channel);
      if (generation !== this.generation) return;
      const subscription = client.feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1 },
      });
      [this.subscription, this.key] = [subscription, key];
      subscription.onChange(() => {
        const snapshot = subscription.value(key);
        this.convert(snapshot);
        // A history change that rewrites the newest sample notifies too, so the value
        // redraws. Consumers that count arrivals compare its appended count.
        const { converted, notified, notifiedStatus } = this;
        const { status } = snapshot;
        if (converted === notified && !problemChanged(notifiedStatus, status)) return;
        [this.notified, this.notifiedStatus] = [converted, status];
        this.notify();
      });
    } catch (e) {
      if (generation !== this.generation) return;
      this.valid = false;
      this.loading = false;
      this.failures.report(e, "Failed to stream channel value");
    }
  }
}

type SelectedChannelProperties = Pick<channel.Payload, "key" | "dataType">;

// A channel with no index draws itself, so the cache reports why it cannot draw.
const fetchChannelProperties = async (
  client: Client,
  ch: channel.Key | channel.Name,
  fetchIndex: boolean,
): Promise<SelectedChannelProperties> => {
  const c = await client.channels.retrieve(ch);
  if (!fetchIndex || c.isIndex || c.index === 0)
    return { key: c.key, dataType: c.dataType };
  return { key: c.index, dataType: DataType.TIMESTAMP };
};

const channelDataPropsZ = z.object({
  channel: z.number().or(z.string()),
  useIndexOfChannel: z.boolean().default(false),
  /** The line's x channel, or zero when x is the index of the line's y channel. */
  xChannel: z.number().or(z.string()).default(0),
  /** The home view: a fixed range, or a span that ends now for a live line. */
  window: windowZ,
  aggregation: aggregationZ.default("min_max"),
});

export type ChannelDataProps = z.input<typeof channelDataPropsZ>;

// The points a line draws before a read gives its own.
const DEFAULT_POINTS = 1000;

const viewRange = ({ lower, upper }: bounds.Bounds): TimeRange =>
  new TimeRange(BigInt(Math.max(Math.floor(lower), 0)), BigInt(Math.ceil(upper)));

/**
 * A series source that draws a channel across the view passed to value(), or across
 * the home view when none is passed. A live line draws its newest samples at full
 * resolution. An xy line draws its home view reduced with decimate.
 */
export class ChannelData
  extends AbstractSource<typeof channelDataPropsZ>
  implements SeriesSource
{
  static readonly TYPE = "channel-data";
  schema = channelDataPropsZ;

  private readonly client: Client | null;
  private readonly now: () => TimeStamp;
  private generation = 0;
  private valid = false;
  private channel: SelectedChannelProperties | null = null;
  // xy is true when the line's x channel is not an index.
  private xy = false;
  private points = DEFAULT_POINTS;
  // draw serves the samples the line draws, and bounds the min_max samples that bound a
  // line that is not xy. Each status is the last one its subscription served.
  private draw: framer.Subscription | null = null;
  private drawStatus?: cstatus.Status;
  private bounds: framer.Subscription | null = null;
  private boundsStatus?: cstatus.Status;
  // The view, points, and settling of the draw's last request, and the points of the
  // bounds' last request. An equal read sends no request.
  private drawView?: bounds.Bounds;
  private drawPoints = 0;
  private drawSettling = false;
  private boundsPoints = 0;
  // boundsRead is true when a bounds read ran since the last draw.
  private boundsRead = false;
  // drawn holds the series of the last draw snapshot. The line buffers them to the GPU,
  // and releasing a series' last reference frees its buffer.
  private drawn = new MultiSeries();
  // loading is true until the channel resolves or its retrieval fails.
  private loading = true;
  private readonly failures = new FailureReporter(() => this.notify());
  private readonly retryNotifier = new sync.Notifier();
  private readonly breaker: breaker.Breaker;

  constructor(
    client: Client | null,
    props: unknown,
    now: () => TimeStamp = client?.now ?? (() => TimeStamp.now()),
    breakerConfig?: breaker.Config,
  ) {
    super(props);
    this.client = client;
    this.now = now;
    this.breaker = framer.retry.createBreaker(this.retryNotifier, breakerConfig);
  }

  private get live(): boolean {
    return this.props.window.variant === "dynamic";
  }

  value({ bounds: view, points, extremesKept = false }: ValueProps = {}): Result<
    [bounds.Bounds, MultiSeries]
  > {
    if (points != null) this.points = points;
    if (primitive.isZero(this.props.channel))
      return success([bounds.INVALID, new MultiSeries()]);
    if (!this.valid) void this.resolve();
    const { client, channel: ch } = this;
    if (client == null || ch == null)
      return {
        data: [bounds.INVALID, new MultiSeries()],
        status: sourceStatus(client, this.loading, this.failures),
        aggregation: this.aggregation(),
      };
    const { series, time } =
      extremesKept && !this.xy
        ? this.readBounds(client, ch.key)
        : this.readDraw(client, ch.key, view, points != null);
    return {
      data: [this.dataBounds(ch, series, time), series],
      status: this.status(),
      aggregation: this.aggregation(),
    };
  }

  // A read with points draws its view. Bounds with no points come while a zoom settles,
  // so they draw the view from the samples held. Any other read serves what the line
  // drew last.
  private readDraw(
    client: Client,
    key: channel.Key,
    view: bounds.Bounds | undefined,
    draws: boolean,
  ): framer.Snapshot {
    const settling = !draws && view != null;
    const target = draws || settling ? view : undefined;
    if (
      this.draw == null ||
      ((draws || settling) && this.drawChanged(target, settling))
    ) {
      const request = this.drawRequest(key, target, settling);
      if (this.draw == null) this.draw = this.subscribe(client, request);
      else this.draw.update(request);
      this.drawView = target && { ...target };
      this.drawPoints = this.points;
      this.drawSettling = settling;
    }
    // A line that stops reading its bounds closes them at its next draw.
    if (draws) {
      if (!this.boundsRead) this.closeBounds();
      this.boundsRead = false;
    }
    const snapshot = this.draw.value(key);
    this.drawStatus = snapshot.status;
    this.hold(snapshot.series);
    return snapshot;
  }

  private drawChanged(view: bounds.Bounds | undefined, settling: boolean): boolean {
    const last = this.drawView;
    return (
      settling !== this.drawSettling ||
      this.points !== this.drawPoints ||
      view?.lower !== last?.lower ||
      view?.upper !== last?.upper
    );
  }

  private drawRequest(
    key: channel.Key,
    view: bounds.Bounds | undefined,
    settling: boolean,
  ): framer.PointsRequest {
    const { window, aggregation } = this.props;
    if (this.xy)
      return {
        keys: [key],
        window,
        limit: { variant: "points", count: this.points, aggregation: "decimate" },
      };
    const limit: framer.PointsLimit = {
      variant: "points",
      count: this.points,
      aggregation,
    };
    if (view != null && bounds.isFinite(view) && bounds.span(view) > 0)
      limit.view = viewRange(view);
    if (settling) limit.settling = true;
    return { keys: [key], window, limit };
  }

  // Reads the samples that bound the home view, reduced with min_max so they keep the
  // extremes that other aggregations drop. The draw serves until the first ones load.
  private readBounds(client: Client, key: channel.Key): framer.Snapshot {
    this.boundsRead = true;
    if (this.bounds == null || this.points !== this.boundsPoints) {
      const request: framer.PointsRequest = {
        keys: [key],
        window: this.props.window,
        limit: {
          variant: "points",
          count: this.points,
          aggregation: this.props.aggregation,
          extremesKept: true,
        },
      };
      if (this.bounds == null) this.bounds = this.subscribe(client, request);
      else this.bounds.update(request);
      this.boundsPoints = this.points;
    }
    let snapshot = this.bounds.value(key);
    if (snapshot.status.variant === "loading" && snapshot.series.length === 0)
      snapshot = this.draw?.value(key) ?? snapshot;
    this.boundsStatus = snapshot.status;
    return snapshot;
  }

  private closeBounds(): void {
    this.bounds?.close();
    this.bounds = null;
    this.boundsStatus = undefined;
  }

  private subscribe(
    client: Client,
    request: framer.PointsRequest,
  ): framer.Subscription {
    const subscription = client.feed.subscribe(request);
    subscription.onChange(() => this.notify());
    return subscription;
  }

  private hold(series: MultiSeries): void {
    if (series === this.drawn) return;
    series.series.forEach((s) => s.acquire());
    this.drawn.series.forEach((s) => s.release());
    this.drawn = series;
  }

  private status(): cstatus.Status {
    return cstatus.worst(this.drawStatus, this.boundsStatus) ?? LOADING_STATUS;
  }

  private aggregation(): Aggregation {
    return this.xy ? "decimate" : this.props.aggregation;
  }

  private dataBounds(
    ch: SelectedChannelProperties,
    series: MultiSeries,
    time: TimeStamp | undefined,
  ): bounds.Bounds {
    if (!ch.dataType.equals(DataType.TIMESTAMP) || this.xy) return series.bounds;
    const home = resolveWindow(this.props.window, this.now());
    if (!this.live) return home.numericBounds;
    const upper = Number((time ?? home.end).valueOf());
    return { lower: upper - Number(home.span.valueOf()), upper };
  }

  /**
   * Never rejects. A connectivity failure retries under the breaker, and a definitive
   * rejection parks the source. Each new failure notifies the source's listeners.
   */
  private async resolve(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) return;
    const { channel, useIndexOfChannel, xChannel } = this.props;
    while (generation === this.generation)
      try {
        const ch = await fetchChannelProperties(client, channel, useIndexOfChannel);
        const xy =
          !primitive.isZero(xChannel) &&
          !(await client.channels.retrieve(xChannel)).isIndex;
        if (generation !== this.generation) return;
        [this.channel, this.xy] = [ch, xy];
        this.breaker.reset();
        this.failures.reset();
        return this.notify();
      } catch (e) {
        if (generation !== this.generation) return;
        // A failure ends the load, so the status shows it while retries run.
        this.loading = false;
        this.failures.report(e, "Failed to read channel data");
        if (framer.retry.isDefinitive(e) || !(await this.breaker.wait())) return;
      }
  }

  cleanup(): void {
    this.generation++;
    this.retryNotifier.notify();
    this.draw?.close();
    this.draw = null;
    this.drawStatus = undefined;
    this.closeBounds();
    this.boundsRead = false;
    this.hold(new MultiSeries());
    this.valid = false;
    this.channel = null;
    this.loading = true;
    this.failures.reset();
  }
}

type Create = (client: Client | null, props: unknown) => Telem;

const REGISTRY: Record<string, Create> = {
  [ChannelData.TYPE]: (client, props) => new ChannelData(client, props),
  [ChannelValue.NUMBER_TYPE]: (client, props) =>
    new ChannelValue(client, props, NUMBER),
  [ChannelValue.STRING_TYPE]: (client, props) =>
    new ChannelValue(client, props, STRING),
};

export class RemoteFactory {
  type = "remote";
  private readonly client: Client | null;
  constructor(client: Client | null) {
    this.client = client;
  }

  create(spec: Spec): Telem | null {
    return REGISTRY[spec.type]?.(this.client, spec.props) ?? null;
  }
}

export const channelData = (props: ChannelDataProps): SeriesSourceSpec => ({
  type: ChannelData.TYPE,
  props,
  variant: "source",
  valueType: "series",
});

export const streamChannelValue = (
  props: Omit<StreamChannelValueProps, "units">,
): NumberSourceSpec => ({
  type: ChannelValue.NUMBER_TYPE,
  props,
  variant: "source",
  valueType: "number",
});

export const streamChannelStringValue = (
  props: StreamChannelValueProps,
): StringSourceSpec => ({
  type: ChannelValue.STRING_TYPE,
  props,
  variant: "source",
  valueType: "string",
});
