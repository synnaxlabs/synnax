// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { unzipSync } from "fflate";
import { describe, expect, it } from "vitest";

import { zip } from "@/zip";

const collect = async (stream: ReadableStream<Uint8Array>): Promise<Uint8Array> =>
  new Uint8Array(await new Response(stream).arrayBuffer());

const source = (path: string, contents: string): zip.Source => ({
  path,
  read: async () => new TextEncoder().encode(contents),
});

describe("create", () => {
  it("archives each source under its path", async () => {
    const bytes = await collect(
      zip.create([source("a.json", "{}"), source("nested/b.json", "[]")]),
    );
    const entries = unzipSync(bytes);
    expect(Object.keys(entries).toSorted()).toEqual(["a.json", "nested/b.json"]);
    expect(new TextDecoder().decode(entries["a.json"])).toBe("{}");
    expect(new TextDecoder().decode(entries["nested/b.json"])).toBe("[]");
  });

  it("reads Blob sources", async () => {
    const bytes = await collect(
      zip.create([{ path: "a.txt", read: async () => new Blob(["blob body"]) }]),
    );
    expect(new TextDecoder().decode(unzipSync(bytes)["a.txt"])).toBe("blob body");
  });

  it("archives nothing into a valid empty zip", async () => {
    const entries = unzipSync(await collect(zip.create([])));
    expect(Object.keys(entries)).toHaveLength(0);
  });

  it("errors the stream when a source read fails", async () => {
    const failing: zip.Source = {
      path: "a.json",
      read: async () => {
        throw new Error("disk gone");
      },
    };
    await expect(collect(zip.create([failing]))).rejects.toThrow("disk gone");
  });
});
