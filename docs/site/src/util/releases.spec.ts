// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import {
  assetURL,
  highest,
  manifestURL,
  type Release,
  Releases,
  tag,
} from "@/util/releases";

const release = (tag_name: string, prerelease = false, draft = false): Release => ({
  tag_name,
  prerelease,
  draft,
});

const RELEASES: Release[] = [
  release("console/v0.58.3"),
  release("console/v0.59.0-rc.2", true),
  release("console/v0.59.0-rc.1", true),
  release("core/v0.58.2"),
  release("driver/v0.59.0"),
  release("synnax-v0.58.2"),
  release("console-v0.58.2"),
  release("console/v0.60.0", false, true),
];

describe("releases", () => {
  describe("tag and URLs", () => {
    it("should build the product tag", () => {
      expect(tag("core", "0.59.0")).toEqual("core/v0.59.0");
    });

    it("should point assets and the manifest at the product release", () => {
      expect(assetURL("driver", "0.59.1", "install-driver-nilinuxrt.sh")).toEqual(
        "https://github.com/synnaxlabs/synnax/releases/download/driver/v0.59.1/install-driver-nilinuxrt.sh",
      );
      expect(manifestURL("0.59.0")).toEqual(
        "https://github.com/synnaxlabs/synnax/releases/download/console/v0.59.0/latest.json",
      );
    });
  });

  describe("highest", () => {
    it("should skip candidates on the stable channel", () => {
      expect(highest(RELEASES, "console", "stable")).toEqual("0.58.3");
    });

    it("should rank the newest candidate on the next channel", () => {
      expect(highest(RELEASES, "console", "next")).toEqual("0.59.0-rc.2");
    });

    it("should rank a stable above its candidates", () => {
      const releases = [...RELEASES, release("console/v0.59.0")];
      expect(highest(releases, "console", "next")).toEqual("0.59.0");
    });

    it("should order by semver, not by listing position", () => {
      const releases = [release("core/v0.58.5"), release("core/v0.59.0")];
      expect(highest(releases, "core", "stable")).toEqual("0.59.0");
      expect(highest([...releases].reverse(), "core", "stable")).toEqual("0.59.0");
    });

    it("should treat the API pre-release flag like a candidate tag", () => {
      const releases = [release("core/v0.59.0", true), release("core/v0.58.2")];
      expect(highest(releases, "core", "stable")).toEqual("0.58.2");
    });

    it("should ignore drafts, legacy tags, and other products", () => {
      expect(highest(RELEASES, "core", "stable")).toEqual("0.58.2");
      expect(highest(RELEASES, "driver", "stable")).toEqual("0.59.0");
      expect(highest([release("synnax-v0.58.2")], "core", "next")).toBeNull();
    });
  });

  describe("Releases", () => {
    const respond = (body: unknown, ok = true, status = 200, next?: string): Response =>
      ({
        ok,
        status,
        json: async () => body,
        headers: new Headers(next == null ? {} : { link: `<${next}>; rel="next"` }),
      }) as unknown as Response;

    it("should fetch once per TTL window", async () => {
      let time = 0;
      const fetch = vi.fn(async () => respond(RELEASES));
      const releases = new Releases({
        fetch,
        ttl: TimeSpan.minutes(5),
        now: () => time,
      });
      expect(await releases.latest("console")).toEqual("0.58.3");
      expect(await releases.latest("driver", "next")).toEqual("0.59.0");
      expect(fetch).toHaveBeenCalledTimes(1);
      time = TimeSpan.minutes(5).milliseconds;
      expect(await releases.latest("core")).toEqual("0.58.2");
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should send the token", async () => {
      const fetch = vi.fn(async () => respond(RELEASES));
      await new Releases({ fetch, token: "abc" }).latest("core");
      const [, init] = fetch.mock.calls[0] as unknown as [string, RequestInit];
      expect(init.headers).toMatchObject({ Authorization: "Bearer abc" });
    });

    it("should throw on an API error and retry on the next lookup", async () => {
      const fetch = vi
        .fn()
        .mockResolvedValueOnce(respond(null, false, 503))
        .mockResolvedValueOnce(respond(RELEASES));
      const releases = new Releases({ fetch });
      await expect(releases.latest("core")).rejects.toThrow("503");
      expect(await releases.latest("core")).toEqual("0.58.2");
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should follow pagination while pages list product releases", async () => {
      const fetch = vi
        .fn()
        .mockResolvedValueOnce(respond([release("core/v0.59.0")], true, 200, "p2"))
        .mockResolvedValueOnce(respond([release("core/v0.59.1")], true, 200, "p3"))
        .mockResolvedValueOnce(respond([release("synnax-v0.58.2")], true, 200, "p4"));
      const releases = new Releases({ fetch });
      expect(await releases.latest("core")).toEqual("0.59.1");
      expect(fetch).toHaveBeenCalledTimes(3);
      expect(fetch.mock.calls[1]?.[0]).toEqual("p2");
      expect(fetch.mock.calls[2]?.[0]).toEqual("p3");
    });

    it("should stop at the last page", async () => {
      const fetch = vi.fn().mockResolvedValueOnce(respond([release("core/v0.59.0")]));
      expect(await new Releases({ fetch }).latest("core")).toEqual("0.59.0");
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should throw when the product has no release", async () => {
      const releases = new Releases({ fetch: vi.fn(async () => respond([])) });
      await expect(releases.latest("driver")).rejects.toThrow("no stable driver");
    });
  });
});
