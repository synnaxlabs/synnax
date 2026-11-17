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
  appVersion,
  assetURL,
  highest,
  manifestURL,
  type Release,
  Releases,
  tag,
} from "@/util/releases";

const release = (
  tag_name: string,
  prerelease = false,
  draft = false,
  published_at: string | null = null,
): Release => ({ tag_name, prerelease, draft, published_at });

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
      expect(manifestURL("console", "0.59.0")).toEqual(
        "https://github.com/synnaxlabs/synnax/releases/download/console/v0.59.0/latest.json",
      );
      expect(manifestURL("desktop", "1.2.0")).toEqual(
        "https://github.com/synnaxlabs/synnax/releases/download/desktop/v1.2.0/latest.json",
      );
    });
  });

  describe("appVersion", () => {
    it("should keep a stable version", () => {
      expect(appVersion("1.2.0")).toEqual("1.2.0");
    });

    it("should make a candidate's pre-release numeric", () => {
      expect(appVersion("1.0.0-rc.3")).toEqual("1.0.0-3");
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

    it("should rank Desktop apart from the train products", () => {
      const releases = [release("desktop/v1.2.0"), ...RELEASES];
      expect(highest(releases, "desktop", "stable")).toEqual("1.2.0");
      expect(highest(releases, "console", "stable")).toEqual("0.58.3");
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

  describe("highest on a train", () => {
    const releases = [
      ...RELEASES,
      release("console/v0.59.3"),
      release("console/v0.60.1"),
    ];

    it("should keep to the train", () => {
      expect(highest(releases, "console", "stable", "0.59")).toEqual("0.59.3");
    });

    it("should return null for a train with no release", () => {
      expect(highest(releases, "console", "stable", "0.57")).toBeNull();
    });
  });

  describe("Releases", () => {
    const respond = (body: unknown, status = 200, next?: string): Response =>
      new Response(JSON.stringify(body), {
        status,
        headers: next == null ? {} : { link: `<${next}>; rel="next"` },
      });

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
      const fetch = vi.fn<typeof globalThis.fetch>(async () => respond(RELEASES));
      await new Releases({ fetch, token: "abc" }).latest("core");
      expect(fetch.mock.calls[0]?.[1]?.headers).toMatchObject({
        Authorization: "Bearer abc",
      });
    });

    it("should throw on an API error and retry on the next lookup", async () => {
      const fetch = vi
        .fn()
        .mockResolvedValueOnce(respond(null, 503))
        .mockResolvedValueOnce(respond(RELEASES));
      const releases = new Releases({ fetch });
      await expect(releases.latest("core")).rejects.toThrow("503");
      expect(await releases.latest("core")).toEqual("0.58.2");
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should follow pagination while pages list product releases", async () => {
      const fetch = vi
        .fn()
        .mockResolvedValueOnce(respond([release("core/v0.59.0")], 200, "p2"))
        .mockResolvedValueOnce(respond([release("core/v0.59.1")], 200, "p3"))
        .mockResolvedValueOnce(respond([release("synnax-v0.58.2")], 200, "p4"));
      const releases = new Releases({ fetch });
      expect(await releases.latest("core")).toEqual("0.59.1");
      expect(fetch).toHaveBeenCalledTimes(3);
      expect(fetch.mock.calls[1]?.[0]).toEqual("p2");
      expect(fetch.mock.calls[2]?.[0]).toEqual("p3");
    });

    describe("publishedAt", () => {
      const PUBLISHED = "2026-10-08T01:18:23Z";

      it("should return when the stable release was published", async () => {
        const fetch = vi.fn(async () =>
          respond([release("core/v0.59.0", false, false, PUBLISHED)]),
        );
        expect(await new Releases({ fetch }).publishedAt("core", "0.59.0")).toEqual(
          new Date(PUBLISHED),
        );
      });

      it("should find a release on a later page", async () => {
        const fetch = vi
          .fn()
          .mockResolvedValueOnce(respond([release("core/v0.61.0")], 200, "p2"))
          .mockResolvedValueOnce(
            respond([release("core/v0.59.0", false, false, PUBLISHED)], 200, "p3"),
          )
          .mockResolvedValueOnce(respond([release("synnax-v0.58.2")]));
        expect(await new Releases({ fetch }).publishedAt("core", "0.59.0")).toEqual(
          new Date(PUBLISHED),
        );
      });

      it("should return null for a version with no release", async () => {
        const fetch = vi.fn(async () => respond(RELEASES));
        expect(await new Releases({ fetch }).publishedAt("core", "0.62.0")).toBeNull();
      });

      it("should return null for a draft, a candidate, and a pre-release", async () => {
        const fetch = vi.fn(async () =>
          respond([
            release("core/v0.62.0", false, true),
            release("core/v0.63.0-rc.1", false, false, PUBLISHED),
            release("core/v0.64.0", true, false, PUBLISHED),
          ]),
        );
        const releases = new Releases({ fetch });
        expect(await releases.publishedAt("core", "0.62.0")).toBeNull();
        expect(await releases.publishedAt("core", "0.63.0-rc.1")).toBeNull();
        expect(await releases.publishedAt("core", "0.64.0")).toBeNull();
      });

      it("should return null for another product's release", async () => {
        const fetch = vi.fn(async () =>
          respond([release("console/v0.59.0", false, false, PUBLISHED)]),
        );
        expect(await new Releases({ fetch }).publishedAt("core", "0.59.0")).toBeNull();
      });
    });

    it("should stop at the last page", async () => {
      const fetch = vi.fn().mockResolvedValueOnce(respond([release("core/v0.59.0")]));
      expect(await new Releases({ fetch }).latest("core")).toEqual("0.59.0");
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should keep documented lookups to the train", async () => {
      const listing = [
        release("core/v0.59.3"),
        release("core/v0.60.1"),
        release("desktop/v1.0.4"),
      ];
      const releases = new Releases({
        fetch: vi.fn(async () => respond(listing)),
        train: "0.59",
      });
      expect(await releases.documented("core")).toEqual("0.59.3");
      expect(await releases.documented("desktop")).toEqual("1.0.4");
      expect(await releases.latest("core")).toEqual("0.60.1");
      expect(await releases.onTrain("core", "0.60")).toEqual("0.60.1");
      expect(await releases.trains("core")).toEqual(["0.60", "0.59"]);
    });

    it("should list each train with a stable release once, newest first", async () => {
      const listing = [
        release("core/v0.59.3"),
        release("core/v0.60.1"),
        release("core/v0.59.0"),
        release("core/v0.61.0-rc.1", true),
        release("core/v0.60.0"),
        release("console/v0.62.0"),
      ];
      const releases = new Releases({ fetch: vi.fn(async () => respond(listing)) });
      expect(await releases.trains("core")).toEqual(["0.60", "0.59"]);
    });

    it("should take the newest release without a train", async () => {
      const listing = [release("core/v0.59.3"), release("core/v0.60.1")];
      const releases = new Releases({ fetch: vi.fn(async () => respond(listing)) });
      expect(await releases.documented("core")).toEqual("0.60.1");
    });

    it("should throw when the train has no documented release", async () => {
      const releases = new Releases({
        fetch: vi.fn(async () => respond([release("core/v0.60.1")])),
        train: "0.59",
      });
      await expect(releases.documented("core")).rejects.toThrow(
        "GitHub lists no stable core release on train 0.59",
      );
    });

    it("should throw when the product has no release", async () => {
      const releases = new Releases({ fetch: vi.fn(async () => respond([])) });
      await expect(releases.latest("driver")).rejects.toThrow("no stable driver");
    });
  });
});
