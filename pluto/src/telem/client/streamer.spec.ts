// // Copyright 2025 Synnax Labs, Inc.
// //
// // Use of this software is governed by the Business Source License included in the file
// // licenses/BSL.txt.
// //
// // As of the Change Date specified in that file, in accordance with the Business Source
// // License, use of this software will be governed by the Apache License, Version 2.0,
// // included in the file licenses/APL.txt.

// import { type channel, Frame, type framer } from "@synnaxlabs/client";
// import { Series, sleep, TimeSpan } from "@synnaxlabs/x";
// import { describe, expect, it, vi } from "vitest";

// import { Cache } from "@/telem/client/cache/cache";
// import { MockRetriever } from "@/telem/client/reader.spec";
// import { Streamer } from "@/telem/client/streamer";
// import { type ReadResponse } from "@/telem/client/types";

// class MockStreamer implements CoreStreamer {
//   private keysI: channel.Keys[];
//   readonly updateVi = vi.fn();
//   readonly closeVi = vi.fn();
//   readonly iteratorVi = vi.fn();
//   readonly nextVi = vi.fn();
//   readonly reads?: framer.Frame[];
//   readonly nextFn?: () => Promise<IteratorResult<framer.Frame>>;

//   constructor(
//     keys: channel.Keys,
//     nextFn?: () => Promise<IteratorResult<framer.Frame>>,
//     reads?: framer.Frame[],
//   ) {
//     this.keysI = [keys];
//     this.reads = reads;
//     this.nextFn = nextFn;
//   }

//   get keys(): channel.Keys {
//     return this.keysI.at(-1) as channel.Keys;
//   }

//   update(keys: channel.Keys): Promise<void> {
//     this.keysI.push(keys);
//     this.updateVi();
//     return Promise.resolve();
//   }

//   close(): void {
//     this.closeVi();
//   }

//   async next(): Promise<IteratorResult<framer.Frame>> {
//     if (this.reads == null && this.nextFn == null)
//       throw new Error("no next function or reads provided");
//     if (this.nextFn != null) return await this.nextFn();
//     const fr = this.reads?.shift();
//     this.nextVi(fr);
//     if (fr == null) return { done: true, value: undefined };
//     return { done: false, value: fr };
//   }

//   [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
//     this.iteratorVi();
//     return this;
//   }
// }

// class MockClient implements StreamClient {
//   private readonly streamers: MockStreamer[];

//   constructor(streamers: MockStreamer[]) {
//     this.streamers = streamers;
//   }

//   openStreamer(keys: channel.Keys): Promise<CoreStreamer> {
//     return Promise.resolve(this.streamers.shift() as CoreStreamer);
//   }
// }

// describe("Streamer", () => {
//   describe("construction", () => {
//     it("should correctly construct a new streamer that operates", async () => {
//       const streamer = new Streamer({
//         cache: new Cache({ channelRetriever: new MockRetriever() }),
//         core: new MockClient([]),
//       });
//       expect(streamer).toBeDefined();
//     });
//   });

//   describe("basic operation", () => {
//     it("should allow the caller to subscribe to changes from a channel", async () => {
//       const retriever = new MockRetriever();
//       let i = 0;
//       const streamer = new Streamer({
//         cache: new Cache({ channelRetriever: retriever }),
//         core: new MockClient([
//           new MockStreamer([1], async () => {
//             await sleep.sleep(TimeSpan.milliseconds(5));
//             i++;
//             return {
//               done: false,
//               value: new Frame({
//                 1: new Series({
//                   data: new Float32Array([1]),
//                   alignment: BigInt(i),
//                 }),
//               }),
//             };
//           }),
//         ]),
//       });

//       const responses: Record<channel.Key, ReadResponse>[] = [];
//       const disconnect = await streamer.stream((d) => responses.push(d), [1]);
//       await expect.poll(() => responses.length > 5).toBeTruthy();
//       await disconnect();

//       // We should only ever get data for that particular channel.
//       expect(responses.every((r) => Object.keys(r).length === 1)).toBeTruthy();
//       // The first response should have no data, as it's just pulling initial relevant
//       // values from the cache.
//       expect(responses[0][1].data.length).toEqual(0);
//       // We should only every has a single response that has data, as its the only
//       // buffer we're allocating and subsequent calls just tell the handler to re-read
//       // the buffer.
//       expect(responses.filter((r) => r[1].data.length === 1)).toHaveLength(1);
//     });
//   });
// });
