// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { ENDPOINT, send } from "./contact";

describe("send", () => {
  afterEach(() => vi.unstubAllGlobals());

  const stub = (respond: () => Promise<Response>) => {
    const fetch = vi.fn(respond);
    vi.stubGlobal("fetch", fetch);
    return fetch;
  };

  it("should post the form data to Formspree as JSON", async () => {
    const fetch = stub(async () => new Response(null, { status: 200 }));
    const data = new FormData();
    data.append("name", "Gaal");
    await send(data);
    expect(fetch).toHaveBeenCalledExactlyOnceWith(ENDPOINT, {
      method: "POST",
      body: data,
      headers: { Accept: "application/json" },
    });
  });

  it("should report a message that Formspree accepts", async () => {
    stub(async () => new Response(null, { status: 200 }));
    expect(await send(new FormData())).toBe(true);
  });

  it("should report a message that Formspree refuses", async () => {
    stub(async () => new Response(null, { status: 422 }));
    expect(await send(new FormData())).toBe(false);
  });

  it("should report a message that the network drops", async () => {
    stub(async () => {
      throw new TypeError("Failed to fetch");
    });
    expect(await send(new FormData())).toBe(false);
  });
});
