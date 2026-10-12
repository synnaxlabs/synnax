// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { post, RequestError } from "@/ui/api";

const answer = (res: Response): ReturnType<typeof vi.fn> => {
  const fetch = vi.fn(async () => res);
  vi.stubGlobal("fetch", fetch);
  return fetch;
};

describe("api", () => {
  afterEach(() => vi.unstubAllGlobals());

  describe("post", () => {
    it("should send the body as JSON", async () => {
      const fetch = answer(new Response(null, { status: 204 }));
      await post("/api/licenses", { nodes: 3 });
      expect(fetch).toHaveBeenCalledWith("/api/licenses", {
        method: "POST",
        headers: { accept: "application/json", "content-type": "application/json" },
        body: '{"nodes":3}',
      });
    });

    it("should return nothing on 204", async () => {
      answer(new Response(null, { status: 204 }));
      expect(await post("/api/licenses/abc/revoke")).toBeUndefined();
    });

    it("should return the JSON body", async () => {
      answer(Response.json({ key: "abc" }));
      expect(await post<{ key: string }>("/api/licenses", {})).toEqual({ key: "abc" });
    });

    it("should throw the status and error a failed route answers with", async () => {
      answer(Response.json({ error: "Staff only" }, { status: 403 }));
      const err = await post("/api/licenses", {}).catch((e: unknown) => e);
      expect(err).toBeInstanceOf(RequestError);
      expect(err).toMatchObject({ status: 403, message: "Staff only" });
    });
  });
});
