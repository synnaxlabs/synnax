// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type framer, type status, UnexpectedError } from "@synnaxlabs/client";
import {
  TimeRange,
  TimeSpan,
  type TimeStamp,
  type VideoFrame as EncodedFrame,
} from "@synnaxlabs/x";
import { z } from "zod";

import { telem } from "@/telem/aether";
import { Decoder, type DecoderProps } from "@/video/aether/decoder";
import { type VideoSource, type VideoSourceSpec } from "@/video/aether/telem/types";

const streamVideoPropsZ = z.object({ channel: channel.keyZ });

export type StreamVideoProps = z.input<typeof streamVideoPropsZ>;

// The snapshot and the count of appended samples that the frames were decoded up to.
interface Cursor {
  snapshot: framer.Snapshot;
  appended: number;
}

/**
 * Streams the frames of a video channel and decodes them in order. Its value is the
 * newest decoded picture, which stays open until the next picture or cleanup.
 */
export class StreamVideo
  extends telem.AbstractSource<typeof streamVideoPropsZ>
  implements VideoSource
{
  static readonly TYPE = "stream-video";
  schema = streamVideoPropsZ;

  private readonly decoder: Decoder;
  private readonly streamFailures: telem.FailureReporter;
  private readonly decodeFailures: telem.FailureReporter;
  private subscription: framer.Subscription | null = null;
  private cursor: Cursor | null = null;
  private picture: VideoFrame | null = null;
  private status: status.Status = telem.SUCCESS_STATUS;
  private subscribed = false;

  constructor(
    private readonly client: telem.Client | null,
    props: unknown,
    private readonly options?: telem.CreateOptions,
    createDecoder?: DecoderProps["createDecoder"],
  ) {
    super(props);
    this.streamFailures = new telem.FailureReporter(options?.onStatusChange);
    this.decodeFailures = new telem.FailureReporter(options?.onStatusChange);
    this.decoder = new Decoder({
      onPicture: (picture) => {
        this.picture?.close();
        this.picture = picture;
        this.decodeFailures.reset();
        this.notify();
      },
      onError: (e) => this.decodeFailures.report(e, "Failed to decode video"),
      createDecoder,
    });
  }

  value(): telem.Result<VideoFrame | null> {
    if (!this.subscribed) this.subscribe();
    return { data: this.picture, status: this.status };
  }

  cleanup(): void {
    this.subscription?.close();
    this.subscription = null;
    this.decoder.close();
    this.picture?.close();
    this.picture = null;
    this.cursor = null;
    this.status = telem.SUCCESS_STATUS;
    this.subscribed = false;
    this.streamFailures.reset();
    this.decodeFailures.reset();
  }

  private subscribe(): void {
    this.subscribed = true;
    const { client } = this;
    if (client == null) {
      this.options?.onStatusChange?.(telem.DISCONNECTED_STATUS);
      return;
    }
    const subscription = client.feed.subscribe({
      keys: [this.props.channel],
      limit: { variant: "latest", count: 1 },
    });
    this.subscription = subscription;
    this.status = telem.LOADING_STATUS;
    subscription.onChange(() => this.pull(subscription));
  }

  // Decodes the frames that the channel appended since the cursor, which are the newest
  // samples of the snapshot. The frames wait while the channel loads.
  private pull(subscription: framer.Subscription): void {
    const { channel } = this.props;
    const snapshot = subscription.value(channel);
    const statusChanged = snapshot.status !== this.status;
    this.status = snapshot.status;
    this.streamFailures.observe(snapshot.status);
    if (snapshot.status.variant !== "loading" && this.cursor?.snapshot !== snapshot) {
      const { series, appended } = snapshot;
      if (appended == null)
        throw new UnexpectedError(
          `snapshot of channel ${channel} has no appended count`,
        );
      let skip = Math.max(0, series.length - appended + (this.cursor?.appended ?? 0));
      for (const s of series.series) {
        // Video series yield decoded frames, which the generic sample type cannot name.
        const frames = s as unknown as Iterable<EncodedFrame>;
        let i = 0;
        for (const frame of frames) {
          if (i >= skip) this.decoder.decode(frame, s.alignment + BigInt(i));
          i++;
        }
        skip = Math.max(0, skip - s.length);
      }
      this.cursor = { snapshot, appended };
    }
    if (statusChanged) this.notify();
  }
}

export const streamVideo = (props: StreamVideoProps): VideoSourceSpec => ({
  type: StreamVideo.TYPE,
  props,
  variant: "source",
  valueType: "video",
});

const playVideoPropsZ = z.object({ channel: channel.keyZ });

export type PlayVideoProps = z.input<typeof playVideoPropsZ>;

// The span of video the subscription holds from the time shown.
const WINDOW = TimeSpan.seconds(10);
// The subscription moves once the time shown comes this close to the end of its span.
const REFILL = TimeSpan.seconds(4);
// How far past the time shown the source decodes.
const AHEAD = TimeSpan.seconds(1).valueOf();
// A time this far past the next frame to decode seeks instead of decoding on.
const SEEK_GAP = TimeSpan.seconds(2).valueOf();
// The most frames the source holds in the decoder and in decoded pictures.
const MAX_PENDING = 16;

interface Timed {
  frame: EncodedFrame;
  // Nanoseconds since the Unix epoch.
  time: bigint;
}

const micros = (ns: bigint): number => Number(ns / 1000n);

// Pairs each frame of the snapshot with its index sample, which has its alignment.
const pair = (snapshot: framer.Snapshot): Timed[] => {
  const times = snapshot.timestamps?.series ?? [];
  const out: Timed[] = [];
  let t = 0;
  for (const s of snapshot.series.series) {
    // Video series yield decoded frames, which the generic sample type cannot name.
    const frames = s as unknown as Iterable<EncodedFrame>;
    let alignment = s.alignment;
    for (const frame of frames) {
      while (t < times.length && times[t].alignmentBounds.upper <= alignment) t++;
      const ts = times[t];
      if (ts != null && ts.alignment <= alignment)
        out.push({ frame, time: BigInt(ts.data[Number(alignment - ts.alignment)]) });
      alignment++;
    }
  }
  return out;
};

/**
 * Plays the stored frames of a video channel at the times its caller asks for. Its
 * value is the newest decoded picture at or before the time of the last value call,
 * which stays open until the next picture or cleanup. A time far from the frames it
 * decoded starts again from the keyframe before that time.
 */
export class PlayVideo
  extends telem.AbstractSource<typeof playVideoPropsZ>
  implements VideoSource
{
  static readonly TYPE = "play-video";
  schema = playVideoPropsZ;

  private readonly decoder: Decoder;
  private readonly readFailures: telem.FailureReporter;
  private readonly decodeFailures: telem.FailureReporter;
  private subscription: framer.Subscription | null = null;
  private window: TimeRange | null = null;
  private snapshot: framer.Snapshot | null = null;
  private timed: Timed[] = [];
  // next is the time of the next frame to decode, or null until the source seeks.
  private next: bigint | null = null;
  private sequence = 0n;
  // queue holds the decoded pictures after the time shown, oldest first.
  private queue: VideoFrame[] = [];
  private picture: VideoFrame | null = null;
  private time: TimeStamp | null = null;
  private status: status.Status = telem.SUCCESS_STATUS;

  constructor(
    private readonly client: telem.Client | null,
    props: unknown,
    private readonly options?: telem.CreateOptions,
    createDecoder?: DecoderProps["createDecoder"],
  ) {
    super(props);
    this.readFailures = new telem.FailureReporter(options?.onStatusChange);
    this.decodeFailures = new telem.FailureReporter(options?.onStatusChange);
    this.decoder = new Decoder({
      onPicture: (picture) => {
        this.queue.push(picture);
        this.decodeFailures.reset();
        if (this.promote()) this.notify();
      },
      onError: (e) => this.decodeFailures.report(e, "Failed to decode video"),
      createDecoder,
    });
    if (client == null) options?.onStatusChange?.(telem.DISCONNECTED_STATUS);
  }

  value({ time }: telem.ValueProps = {}): telem.Result<VideoFrame | null> {
    if (time != null) this.show(time);
    return { data: this.picture, status: this.status };
  }

  cleanup(): void {
    this.subscription?.close();
    this.subscription = null;
    this.window = null;
    this.snapshot = null;
    this.timed = [];
    this.reset();
    this.picture?.close();
    this.picture = null;
    this.time = null;
    this.status = telem.SUCCESS_STATUS;
    this.readFailures.reset();
    this.decodeFailures.reset();
  }

  private show(time: TimeStamp): void {
    this.time = time;
    this.cover(time);
    const ns = time.valueOf();
    const shown = this.picture?.timestamp;
    if (
      this.next == null ||
      (shown != null && micros(ns) < shown) ||
      ns - this.next > SEEK_GAP
    )
      this.seek(ns);
    this.decode();
    this.promote();
  }

  // Moves the subscription to hold the time and the span after it.
  private cover(time: TimeStamp): void {
    const { client, window } = this;
    if (client == null) return;
    if (window?.contains(time) === true && time.before(window.end.sub(REFILL))) return;
    const next = new TimeRange(time, time.add(WINDOW));
    this.window = next;
    const request: framer.SubscribeRequest = {
      keys: [this.props.channel],
      window: next,
      limit: { variant: "frames" },
    };
    if (this.subscription != null) return this.subscription.update(request);
    const subscription = client.feed.subscribe(request);
    this.subscription = subscription;
    subscription.onChange(() => this.pull(subscription));
  }

  private pull(subscription: framer.Subscription): void {
    const snapshot = subscription.value(this.props.channel);
    this.status = snapshot.status;
    this.readFailures.observe(snapshot.status);
    if (snapshot !== this.snapshot) {
      this.snapshot = snapshot;
      this.timed = pair(snapshot);
    }
    this.notify();
  }

  // Starts decoding again from the last keyframe at or before ns. With no such frame
  // held yet, it waits for the subscription.
  private seek(ns: bigint): void {
    const from = this.timed.findLast((t) => t.frame.keyframe && t.time <= ns);
    if (from == null) return;
    this.reset();
    this.next = from.time;
  }

  private reset(): void {
    this.decoder.close();
    this.queue.forEach((p) => p.close());
    this.queue = [];
    this.next = null;
  }

  // Feeds the decoder the frames from next to a little past the time shown.
  private decode(): void {
    const { next, time } = this;
    if (next == null || time == null) return;
    const until = time.valueOf() + AHEAD;
    let i = this.timed.findIndex((t) => t.time >= next);
    if (i === -1) return;
    while (
      i < this.timed.length &&
      this.timed[i].time <= until &&
      this.decoder.pending + this.queue.length < MAX_PENDING
    ) {
      const { frame, time: ns } = this.timed[i++];
      this.next = ns + 1n;
      this.decoder.decode(frame, this.sequence++, micros(ns));
    }
  }

  // Shows the newest decoded picture at or before the time shown.
  private promote(): boolean {
    const { time } = this;
    if (time == null) return false;
    const until = micros(time.valueOf());
    let changed = false;
    while (this.queue.length > 0 && this.queue[0].timestamp <= until) {
      this.picture?.close();
      this.picture = this.queue.shift()!;
      changed = true;
    }
    return changed;
  }
}

export const playVideo = (props: PlayVideoProps): VideoSourceSpec => ({
  type: PlayVideo.TYPE,
  props,
  variant: "source",
  valueType: "video",
});
