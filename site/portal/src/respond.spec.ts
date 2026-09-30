// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { type Portal } from "@/portal";
import { download, filename, form, handle } from "@/respond";
import { badRequest, HTTPError, toResponse } from "@/server/errors";
import { body, createAPIContext } from "@/testutil";

const PORTAL = {} as Portal;

describe("respond", () => {
  afterEach(() => vi.restoreAllMocks());

  describe("filename", () => {
    it("should turn the license label into a file name", () => {
      expect(filename("Test rig")).toBe("Test-rig.license");
    });

    it("should collapse and trim runs of other characters", () => {
      expect(filename("  Acme / Rig #2!  ")).toBe("Acme-Rig-2.license");
    });

    it("should fall back to synnax for a label with nothing usable", () => {
      expect(filename("")).toBe("synnax.license");
      expect(filename("///")).toBe("synnax.license");
    });
  });

  describe("download", () => {
    it("should answer with the key as an attachment", async () => {
      const res = download("header.payload.signature", "Test rig");
      expect(res.status).toBe(200);
      expect(res.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      expect(res.headers.get("content-disposition")).toBe(
        'attachment; filename="Test-rig.license"',
      );
      expect(await res.text()).toBe("header.payload.signature");
    });
  });

  describe("handle", () => {
    it("should return the body's response", async () => {
      const res = await handle(async () => new Response(null, { status: 204 }));
      expect(res.status).toBe(204);
    });

    it("should answer an HTTP error with its status and message", async () => {
      const res = await handle(async () => {
        throw badRequest("Nodes must be a whole number of at least 1");
      });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({
        error: "Nodes must be a whole number of at least 1",
      });
    });

    it("should answer any other error with a 500 that hides it", async () => {
      const log = vi.spyOn(console, "error").mockImplementation(() => {});
      const res = await handle(async () => {
        throw new Error("connection reset");
      });
      expect(res.status).toBe(500);
      expect(await body(res)).toEqual({ error: "Internal error" });
      expect(log).toHaveBeenCalledOnce();
    });
  });

  describe("toResponse", () => {
    it("should carry the status of an HTTP error", async () => {
      const res = toResponse(new HTTPError(429, "Too many activations."));
      expect(res.status).toBe(429);
      expect(await body(res)).toEqual({ error: "Too many activations." });
    });

    it("should hide a value that is not an error", async () => {
      vi.spyOn(console, "error").mockImplementation(() => {});
      const res = toResponse("boom");
      expect(res.status).toBe(500);
      expect(await body(res)).toEqual({ error: "Internal error" });
    });
  });

  describe("form", () => {
    it("should read a JSON body into strings", async () => {
      const context = createAPIContext(PORTAL, {
        body: {
          label: "Rig",
          nodes: 2,
          fingerprint: ["a", "b"],
          nested: { x: 1 },
        },
      });
      expect(await form(context)).toEqual({
        label: "Rig",
        nodes: "2",
        fingerprint: "a,b",
        nested: '{"x":1}',
      });
    });

    it("should read a JSON body that is not an object as empty", async () => {
      expect(await form(createAPIContext(PORTAL, { body: "text" }))).toEqual({});
      expect(await form(createAPIContext(PORTAL, { body: null }))).toEqual({});
      expect(await form(createAPIContext(PORTAL, { body: ["a", "b"] }))).toEqual({});
    });

    it("should leave out null fields", async () => {
      const context = createAPIContext(PORTAL, {
        body: { label: "Rig", maxVersion: null },
      });
      expect(await form(context)).toEqual({ label: "Rig" });
    });

    it("should read a posted form", async () => {
      const data = new FormData();
      data.set("name", "Test stand");
      data.set("file", new File(["x"], "key.license"));
      expect(await form(createAPIContext(PORTAL, { body: data }))).toEqual({
        name: "Test stand",
        file: "key.license",
      });
    });
  });
});
