// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Pins the deadline on the streamer's open acknowledgement. The socket is open by
// then, so the keepalive deadline from SY-4740 is not yet armed and cannot cover it.

import { Unreachable, type WebSocketClient } from "@synnaxlabs/freighter";
import { DataType } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { payloadZ } from "@/channel/types.gen";
import { type ChannelRetriever } from "@/framer/adapter";
import { createStreamOpener } from "@/framer/streamer";

const retrieveChannels: ChannelRetriever = async () => [
  payloadZ.parse({ key: 1, name: "silent", dataType: DataType.FLOAT64.toString() }),
];

/** A client whose socket opens and is then never spoken to again. */
const createSilentClient = (): { client: WebSocketClient; closed: () => boolean } => {
  let closed = false;
  const client = {
    withCodec: () => client,
    stream: async () => ({
      send: () => {},
      receive: async () => await new Promise<never>(() => {}),
      received: () => false,
      closeSend: () => {
        closed = true;
      },
    }),
  };
  return { client: client as unknown as WebSocketClient, closed: () => closed };
};

describe("streamer open acknowledgement", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("should reject an acknowledgement that never arrives", async () => {
    const { client, closed } = createSilentClient();
    const open = createStreamOpener(retrieveChannels, client)({ channels: [1] });
    const settled = expect(open).rejects.toThrow(Unreachable);
    await vi.advanceTimersByTimeAsync(31_000);
    await settled;
    // The socket is released rather than left open with nobody holding it.
    expect(closed()).toBe(true);
  });
});
