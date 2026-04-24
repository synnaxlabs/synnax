// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { GROUPS, REGISTRY, variantZ } from "@/schematic/symbol/registry";
import { SYNNAX_DARK, themeZ } from "@/theming/base/theme";

const VARIANTS = variantZ.options;
const mockTheme = themeZ.parse(SYNNAX_DARK);

describe("symbol registry", () => {
  describe("registry integrity", () => {
    it("should have a registry entry for every variant", () => {
      for (const variant of VARIANTS) expect(REGISTRY).toHaveProperty(variant);
    });

    it("should have required fields on every registry entry", () => {
      for (const variant of VARIANTS) {
        const spec = REGISTRY[variant];
        expect(spec.name).toBeDefined();
        expect(spec.key).toBe(variant);
        expect(spec.Form).toBeDefined();
        expect(spec.Symbol).toBeDefined();
        expect(spec.defaultProps).toBeDefined();
        expect(spec.Preview).toBeDefined();
        expect(spec.zIndex).toBeDefined();
      }
    });

    it("should return a non-null object from defaultProps for every variant", () => {
      for (const variant of VARIANTS) {
        const props = REGISTRY[variant].defaultProps(mockTheme);
        expect(props).toBeDefined();
        expect(typeof props).toBe("object");
      }
    });

    it("should have no duplicate keys in variants", () => {
      const seen = new Set<string>();
      for (const variant of VARIANTS) {
        expect(seen.has(variant)).toBe(false);
        seen.add(variant);
      }
    });
  });

  describe("groups", () => {
    it("should have unique group keys", () => {
      const keys = GROUPS.map((g) => g.key);
      expect(new Set(keys).size).toBe(keys.length);
    });

    it("should only reference variants that exist in the registry", () => {
      for (const group of GROUPS)
        for (const symbol of group.symbols) expect(REGISTRY).toHaveProperty(symbol);
    });

    it("should include every variant in at least one group", () => {
      const allGroupedSymbols = new Set(GROUPS.flatMap((g) => g.symbols));
      const customVariants = new Set(["customActuator", "customStatic"]);
      for (const variant of VARIANTS) {
        if (customVariants.has(variant)) continue;
        expect(allGroupedSymbols.has(variant)).toBe(true);
      }
    });
  });

  describe("mediaEmbed", () => {
    it("should be in the registry", () => {
      expect(REGISTRY).toHaveProperty("mediaEmbed");
    });

    it("should be in the containers group", () => {
      const containersGroup = GROUPS.find((g) => g.key === "containers");
      expect(containersGroup).toBeDefined();
      expect(containersGroup!.symbols).toContain("mediaEmbed");
    });

    it("should have searchTerms defined and non-empty", () => {
      const spec = REGISTRY.mediaEmbed;
      expect(spec.searchTerms).toBeDefined();
      expect(spec.searchTerms!.length).toBeGreaterThan(0);
    });

    it("should have searchTerms that include common media terms", () => {
      const terms = REGISTRY.mediaEmbed.searchTerms!;
      expect(terms).toContain("video embed");
      expect(terms).toContain("stream embed");
      expect(terms).toContain("camera embed");
      expect(terms).toContain("image embed");
    });

    it("should have default dimensions of 320x180", () => {
      const props = REGISTRY.mediaEmbed.defaultProps(mockTheme);
      expect(props.dimensions).toEqual({ width: 320, height: 180 });
    });

    it("should have an empty url in default props", () => {
      const props = REGISTRY.mediaEmbed.defaultProps(mockTheme);
      expect(props.url).toBe("");
    });
  });

  describe("iframeEmbed", () => {
    it("should be in the registry", () => {
      expect(REGISTRY).toHaveProperty("iframeEmbed");
    });

    it("should be in the containers group", () => {
      const containersGroup = GROUPS.find((g) => g.key === "containers");
      expect(containersGroup).toBeDefined();
      expect(containersGroup!.symbols).toContain("iframeEmbed");
    });

    it("should have searchTerms that include iframe and dashboard terms", () => {
      const terms = REGISTRY.iframeEmbed.searchTerms!;
      expect(terms).toContain("iframe embed");
      expect(terms).toContain("dashboard embed");
      expect(terms).toContain("grafana embed");
      expect(terms).toContain("widget embed");
    });

    it("should have default dimensions of 320x180", () => {
      const props = REGISTRY.iframeEmbed.defaultProps(mockTheme);
      expect(props.dimensions).toEqual({ width: 320, height: 180 });
    });

    it("should have an empty url and blockCookies true in default props", () => {
      const props = REGISTRY.iframeEmbed.defaultProps(mockTheme);
      expect(props.url).toBe("");
      expect(props.blockCookies).toBe(true);
    });
  });

  describe("pageEmbed", () => {
    it("should not be in the registry", () => {
      expect(REGISTRY).not.toHaveProperty("pageEmbed");
    });

    it("should not be in any group", () => {
      for (const group of GROUPS) expect(group.symbols).not.toContain("pageEmbed");
    });
  });

  describe("search integration", () => {
    it("should find mediaEmbed when searching for 'video'", () => {
      const allSpecs = Object.values(REGISTRY);
      const matchingSpecs = allSpecs.filter(
        (s) =>
          s.name.toLowerCase().includes("video") ||
          (s.searchTerms != null && s.searchTerms.toLowerCase().includes("video")),
      );
      expect(matchingSpecs.some((s) => s.key === "mediaEmbed")).toBe(true);
    });

    it("should find mediaEmbed when searching for 'camera'", () => {
      const allSpecs = Object.values(REGISTRY);
      const matchingSpecs = allSpecs.filter(
        (s) =>
          s.name.toLowerCase().includes("camera") ||
          (s.searchTerms != null && s.searchTerms.toLowerCase().includes("camera")),
      );
      expect(matchingSpecs.some((s) => s.key === "mediaEmbed")).toBe(true);
    });

    it("should find mediaEmbed when searching for 'mjpeg'", () => {
      const allSpecs = Object.values(REGISTRY);
      const matchingSpecs = allSpecs.filter(
        (s) =>
          s.name.toLowerCase().includes("mjpeg") ||
          (s.searchTerms != null && s.searchTerms.toLowerCase().includes("mjpeg")),
      );
      expect(matchingSpecs.some((s) => s.key === "mediaEmbed")).toBe(true);
    });

    it("should find iframeEmbed when searching for 'grafana'", () => {
      const allSpecs = Object.values(REGISTRY);
      const matchingSpecs = allSpecs.filter(
        (s) =>
          s.name.toLowerCase().includes("grafana") ||
          (s.searchTerms != null && s.searchTerms.toLowerCase().includes("grafana")),
      );
      expect(matchingSpecs.some((s) => s.key === "iframeEmbed")).toBe(true);
    });

    it("should find iframeEmbed when searching for 'dashboard'", () => {
      const allSpecs = Object.values(REGISTRY);
      const matchingSpecs = allSpecs.filter(
        (s) =>
          s.name.toLowerCase().includes("dashboard") ||
          (s.searchTerms != null && s.searchTerms.toLowerCase().includes("dashboard")),
      );
      expect(matchingSpecs.some((s) => s.key === "iframeEmbed")).toBe(true);
    });
  });
});
