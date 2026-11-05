// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  DataType,
  framer,
  type status as cstatus,
} from "@synnaxlabs/client";
import { createTestClient, SyntheticCore } from "@synnaxlabs/client/testutil";
import {
  id,
  Rate,
  Series,
  TimeSpan,
  TimeStamp,
  type VideoFrame as EncodedFrame,
} from "@synnaxlabs/x";
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from "vitest";

import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { PlayVideo, StreamVideo } from "@/video/aether/telem/sources";
import { videoTest } from "@/video/aether/testutil";

const { STREAM } = videoTest;
const START = TimeStamp.seconds(1_000_000);
const OPTIONS: Partial<framer.FeedProps> = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
};

// jsdom has no WebCodecs, so the specs decode with a stand-in that turns each chunk
// into a picture at once.
class Chunk {
  constructor(readonly init: EncodedVideoChunkInit) {}
  get timestamp(): number {
    return this.init.timestamp;
  }
}

class Picture {
  closed = false;
  constructor(readonly timestamp: number) {}
  close(): void {
    this.closed = true;
  }
}

const createDecoder = (init: VideoDecoderInit): VideoDecoder => {
  let state: CodecState = "unconfigured";
  const decoder = {
    get state() {
      return state;
    },
    decodeQueueSize: 0,
    configure: () => {
      state = "configured";
    },
    decode: (chunk: Chunk) =>
      init.output(new Picture(chunk.timestamp) as unknown as VideoFrame),
    close: () => {
      state = "closed";
    },
  };
  return decoder as unknown as VideoDecoder;
};

const series = (frames: EncodedFrame[], alignment: number): Series =>
  new Series({ data: frames, dataType: DataType.VIDEO, alignment: BigInt(alignment) });

describe("StreamVideo", () => {
  beforeAll(() => vi.stubGlobal("EncodedVideoChunk", Chunk));
  afterAll(() => vi.unstubAllGlobals());
  const closers: Array<() => Promise<void> | void> = [];
  afterEach(async () => {
    for (const close of closers.splice(0).reverse()) await close();
  });

  const createCore = (): SyntheticCore =>
    new SyntheticCore({ rate: Rate.hz(1), start: START, now: START, derived: true });

  // Opens a source on the virtual channel of the Core, and waits for its stream.
  const open = async (core: SyntheticCore, statuses: cstatus.Crude[] = []) => {
    const client = telemTest.createClient(core, OPTIONS);
    closers.push(async () => await client.feed.close());
    const { key } = core.virtual!;
    const source = new StreamVideo(
      client,
      { channel: key },
      { onStatusChange: (s) => statuses.push(s) },
      createDecoder,
    );
    closers.push(() => source.cleanup());
    expect(source.value().data).toBeNull();
    await expect.poll(() => core.streams(key)).toBe(true);
    const stream = (s: Series): void => core.stream(new framer.Frame(key, s));
    return { source, stream };
  };

  // The decoder numbers the pictures in decode order from zero.
  const newest = (source: StreamVideo): number | undefined =>
    source.value().data?.timestamp;

  it("should hold the newest picture of the frames it streams", async () => {
    const { source, stream } = await open(createCore());
    let changes = 0;
    source.onChange(() => changes++);
    stream(series(STREAM, 0));
    await expect.poll(() => newest(source)).toEqual(7);
    expect(changes).toBeGreaterThanOrEqual(1);
  });

  it("should decode frames that stream in separate series in order", async () => {
    const { source, stream } = await open(createCore());
    STREAM.forEach((f, i) => stream(series([f], i)));
    await expect.poll(() => newest(source)).toEqual(7);
  });

  it("should wait for the next keyframe after a gap in alignment", async () => {
    const { source, stream } = await open(createCore());
    stream(series(STREAM.slice(0, 2), 0));
    stream(series(STREAM.slice(3), 3));
    // Frame 3 has no frame 2 to follow, so frames 0, 1, and 4 to 7 decode.
    await expect.poll(() => newest(source)).toEqual(5);
  });

  it("should close the previous picture when a new one arrives", async () => {
    const { source, stream } = await open(createCore());
    stream(series(STREAM.slice(0, 1), 0));
    await expect.poll(() => newest(source)).toEqual(0);
    const first = source.value().data as unknown as Picture;
    stream(series(STREAM.slice(1, 2), 1));
    await expect.poll(() => newest(source)).toEqual(1);
    expect(first.closed).toBe(true);
  });

  it("should close its picture on cleanup", async () => {
    const { source, stream } = await open(createCore());
    stream(series(STREAM.slice(0, 1), 0));
    await expect.poll(() => newest(source)).toEqual(0);
    const picture = source.value().data as unknown as Picture;
    source.cleanup();
    expect(picture.closed).toBe(true);
  });

  it("should report a frame it cannot decode", async () => {
    const statuses: cstatus.Crude[] = [];
    const { stream } = await open(createCore(), statuses);
    stream(series([{ keyframe: true, codec: "h264", data: new Uint8Array([1]) }], 0));
    await expect
      .poll(() => statuses)
      .toContainEqual(
        expect.objectContaining({
          variant: "error",
          message: "Failed to decode video",
          description: "video keyframe has no SPS",
        }),
      );
  });

  it("should report a disconnected status with no client", () => {
    const statuses: cstatus.Crude[] = [];
    const source = new StreamVideo(
      null,
      { channel: 1 },
      { onStatusChange: (s) => statuses.push(s) },
    );
    expect(source.value().data).toBeNull();
    expect(statuses).toEqual([telem.DISCONNECTED_STATUS]);
  });
});

describe("PlayVideo", () => {
  beforeAll(() => vi.stubGlobal("EncodedVideoChunk", Chunk));
  afterAll(() => vi.unstubAllGlobals());

  const synnax = createTestClient();
  const feed = synnax.openFeed();
  afterAll(async () => await feed.close());
  const client: telem.Client = {
    feed,
    channels: synnax.channels,
    now: () => TimeStamp.now(),
  };

  const at = (seconds: number): TimeStamp => START.add(TimeSpan.seconds(seconds));
  const micros = (ts: TimeStamp): number => Number(ts.valueOf() / 1000n);

  // Writes the frames of STREAM, one each second from START, to a new channel.
  const createVideo = async (): Promise<channel.Channel> => {
    const idx = await synnax.channels.create({
      name: id.create(),
      isIndex: true,
      dataType: DataType.TIMESTAMP,
    });
    const video = await synnax.channels.create({
      name: id.create(),
      index: idx.key,
      dataType: DataType.VIDEO,
    });
    onTestFinished(async () => await synnax.channels.delete([video.key, idx.key]));
    await synnax.write(START, {
      [idx.key]: STREAM.map((_, i) => at(i)),
      [video.key]: new Series({ data: STREAM, dataType: DataType.VIDEO }),
    });
    return video;
  };

  const open = async (): Promise<PlayVideo> => {
    const video = await createVideo();
    const source = new PlayVideo(client, { channel: video.key }, {}, createDecoder);
    onTestFinished(() => source.cleanup());
    return source;
  };

  const shown = (source: PlayVideo, time: TimeStamp): number | undefined =>
    source.value({ time }).data?.timestamp;

  it("should show the newest frame at or before the time", async () => {
    const source = await open();
    await expect.poll(() => shown(source, at(5.5))).toBe(micros(at(5)));
  });

  it("should show later frames as the time moves on", async () => {
    const source = await open();
    await expect.poll(() => shown(source, at(5.5))).toBe(micros(at(5)));
    await expect.poll(() => shown(source, at(7))).toBe(micros(at(7)));
  });

  it("should decode again from the keyframe before an earlier time", async () => {
    const source = await open();
    await expect.poll(() => shown(source, at(6))).toBe(micros(at(6)));
    await expect.poll(() => shown(source, at(1.5))).toBe(micros(at(1)));
  });

  it("should close each picture it replaces", async () => {
    const source = await open();
    await expect.poll(() => shown(source, at(4))).toBe(micros(at(4)));
    const first = source.value().data as unknown as Picture;
    await expect.poll(() => shown(source, at(5))).toBe(micros(at(5)));
    expect(first.closed).toBe(true);
  });

  it("should show nothing before the first frame", async () => {
    const source = await open();
    await expect
      .poll(() => source.value({ time: at(-1) }).status.variant)
      .toBe("success");
    expect(source.value({ time: at(-1) }).data).toBeNull();
  });

  it("should report a disconnected status with no client", () => {
    const statuses: cstatus.Crude[] = [];
    const source = new PlayVideo(
      null,
      { channel: 1 },
      { onStatusChange: (s) => statuses.push(s) },
    );
    expect(source.value({ time: at(0) }).data).toBeNull();
    expect(statuses).toEqual([telem.DISCONNECTED_STATUS]);
  });
});
