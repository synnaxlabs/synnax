// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Runtime } from "@/platform/runtime";

const streamOf = (...chunks: string[]): ReadableStream<Uint8Array> =>
  new ReadableStream({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(new TextEncoder().encode(chunk));
      controller.close();
    },
  });

describe("Runtime.uploadBody", () => {
  it("passes bytes through untouched", async () => {
    const bytes = new TextEncoder().encode("payload");
    await expect(Runtime.uploadBody(bytes)).resolves.toBe(bytes);
  });

  it("passes a Blob through untouched", async () => {
    const blob = new Blob(["payload"]);
    await expect(Runtime.uploadBody(blob)).resolves.toBe(blob);
  });

  it("buffers a stream into a single Blob", async () => {
    const body = await Runtime.uploadBody(streamOf("pay", "load"));
    expect(body).toBeInstanceOf(Blob);
    expect(await (body as Blob).text()).toBe("payload");
  });
});
