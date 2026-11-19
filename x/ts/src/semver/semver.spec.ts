// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { semver } from "@/semver";
import { zod } from "@/zod";

describe("semver", () => {
  describe("versionZ", () => {
    it("should accept a standard version", () => {
      expect(semver.versionZ.parse("1.0.0")).toEqual("1.0.0");
      expect(semver.versionZ.parse("0.0.0")).toEqual("0.0.0");
      expect(semver.versionZ.parse("99.99.99")).toEqual("99.99.99");
    });

    it("should accept pre-release versions", () => {
      [
        "1.0.0-alpha",
        "0.48.0-rc",
        "1.0.0-0",
        "1.0.0-99",
        "1.0.0-alpha.1",
        "1.0.0-beta.2.3",
        "1.0.0-0.3.7",
        "1.0.0-x-beta",
      ].forEach((v) => expect(semver.versionZ.parse(v)).toEqual(v));
    });

    it("should reject a version without a minor or patch", () => {
      expect(semver.versionZ.safeParse("1.0").success).toBe(false);
      expect(semver.versionZ.safeParse("1").success).toBe(false);
    });

    it("should reject build metadata and an empty pre-release", () => {
      expect(semver.versionZ.safeParse("1.0.0+build").success).toBe(false);
      expect(semver.versionZ.safeParse("1.0.0-").success).toBe(false);
    });
  });

  describe("compare", () => {
    it("should order versions by major, minor, and patch", () => {
      expect(semver.compare("1.0.0", "0.9.9")).toBe(1);
      expect(semver.compare("1.1.0", "1.2.0")).toBe(-1);
      expect(semver.compare("1.1.10", "1.1.9")).toBe(1);
      expect(semver.compare("1.1.1", "1.1.1")).toBe(0);
    });

    it("should skip the major version when it is ignored", () => {
      expect(semver.compare("2.1.0", "1.1.0", { majorIgnored: true })).toBe(0);
      expect(semver.compare("2.0.0", "1.1.0", { majorIgnored: true })).toBe(-1);
    });

    it("should skip the minor version when it is ignored", () => {
      expect(semver.compare("1.2.0", "1.1.0", { minorIgnored: true })).toBe(0);
      expect(semver.compare("1.2.0", "1.1.1", { minorIgnored: true })).toBe(-1);
    });

    it("should compare only the major version when the rest is ignored", () => {
      const options = { minorIgnored: true, patchIgnored: true };
      expect(semver.compare("1.0.0", "1.1.0", options)).toBe(0);
      expect(semver.compare("2.0.0", "1.1.0", options)).toBe(1);
      expect(semver.compare("1.0.0", "2.1.0", options)).toBe(-1);
    });

    it("should compare only the patch version when the rest is ignored", () => {
      const options = { majorIgnored: true, minorIgnored: true };
      expect(semver.compare("2.1.1", "2.2.1", options)).toBe(0);
      expect(semver.compare("1.4.2", "1.9.1", options)).toBe(1);
      expect(semver.compare("10000.2.0", "95.6.1", options)).toBe(-1);
    });

    it("should skip the pre-release when the patch is ignored", () => {
      const options = { patchIgnored: true };
      expect(semver.compare("0.56.2", "0.56.16-79dd4e2", options)).toBe(0);
      expect(semver.compare("0.57.0", "0.56.16-79dd4e2", options)).toBe(1);
    });

    it("should not change the options object", () => {
      const options = { patchIgnored: true };
      semver.compare("1.0.0", "1.0.1", options);
      expect(options).toEqual({ patchIgnored: true });
    });

    it("should order a release after its pre-releases", () => {
      expect(semver.compare("1.0.0", "1.0.0-rc")).toBe(1);
      expect(semver.compare("0.48.0-rc", "0.48.0")).toBe(-1);
    });

    it("should order pre-release identifiers by semver precedence", () => {
      const versions = [
        "1.0.0-1",
        "1.0.0-2",
        "1.0.0-10",
        "1.0.0-alpha",
        "1.0.0-alpha.1",
        "1.0.0-alpha.1.2",
        "1.0.0-alpha.beta",
        "1.0.0-beta",
        "1.0.0-beta.2",
        "1.0.0-beta.11",
        "1.0.0-rc",
        "1.0.0-rc.1",
        "1.0.0-rc.beta",
        "1.0.0",
      ];
      for (let i = 0; i < versions.length - 1; i++) {
        expect(semver.compare(versions[i], versions[i + 1])).toBe(-1);
        expect(semver.compare(versions[i + 1], versions[i])).toBe(1);
      }
    });

    it("should compare a pre-release that holds a hyphen in full", () => {
      expect(semver.compare("1.0.0-x-alpha", "1.0.0-x-beta")).toBe(-1);
      expect(semver.compare("1.0.0-x-beta", "1.0.0-x-beta")).toBe(0);
    });

    it("should consider equal pre-releases equal", () => {
      expect(semver.compare("1.0.0-alpha.1", "1.0.0-alpha.1")).toBe(0);
    });

    it("should sort an array of versions", () => {
      expect(["1.0.0", "0.0.0", "1.0.0-rc"].sort(semver.compare)).toEqual([
        "0.0.0",
        "1.0.0-rc",
        "1.0.0",
      ]);
    });

    it("should throw a parse error for a value that is not a version", () => {
      expect(() => semver.compare("1.0", "1.0.0")).toThrow(zod.ParseError);
    });
  });

  describe("equal", () => {
    it("should return true only for equal versions", () => {
      expect(semver.equal("1.0.0-rc", "1.0.0-rc")).toBe(true);
      expect(semver.equal("1.0.0", "1.0.1")).toBe(false);
      expect(semver.equal("0.56.2", "0.56.16", { patchIgnored: true })).toBe(true);
    });
  });

  describe("newer", () => {
    it("should return true only when the first version is newer", () => {
      expect(semver.newer("3.0.0", "0.3.0")).toBe(true);
      expect(semver.newer("1.0.0-rc", "1.0.0-alpha")).toBe(true);
      expect(semver.newer("1.0.0", "1.0.0")).toBe(false);
      expect(semver.newer("0.56.3", "0.56.2", { patchIgnored: true })).toBe(false);
    });
  });

  describe("older", () => {
    it("should return true only when the first version is older", () => {
      expect(semver.older("1.0.0-rc", "1.0.0")).toBe(true);
      expect(semver.older("1.0.0-alpha", "1.0.0-beta")).toBe(true);
      expect(semver.older("1.0.0", "1.0.0")).toBe(false);
    });
  });
});
