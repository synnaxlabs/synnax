// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { type ComponentProps } from "astro/types";
import { type Element, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { beforeAll, describe, expect, it } from "vitest";

import { Hero } from "@/components/hero";
import linePlots from "@/components/hero/stills/console/line-plots/hero.json";
import schematics from "@/components/hero/stills/console/schematics/hero.json";
import { CDN } from "@/components/media/cdn";

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const withClass = (node: Root | Element, name: string): Element[] =>
  elements(node).filter(({ properties: { className } }) =>
    Array.isArray(className) ? className.includes(name) : false,
  );

/** The classes of the hero's root element. */
const classes = (root: Root): unknown =>
  withClass(root, "docs-hero")[0].properties.className;

const images = (node: Root | Element): unknown[] =>
  elements(node)
    .filter((child) => child.tagName === "img")
    .map((img) => img.properties.src);

let container: AstroContainer;

beforeAll(async () => {
  container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
});

const render = async (props: ComponentProps<typeof Hero.Hero>): Promise<Root> =>
  fromHtml(await container.renderToString(Hero.Hero, { props }), { fragment: true });

describe("Hero.Hero", () => {
  describe("still", () => {
    it("should show the version of the still that Studio recorded", async () => {
      const root = await render({
        still: "console/schematics/hero",
        layout: "bleed",
        tone: "deep",
        palette: "violet-pink",
      });
      expect(images(root)).toEqual([
        CDN.url(
          `console/schematics/hero/${schematics.version}/window`,
          "webp",
          "light",
        ),
      ]);
      expect(classes(root)).toEqual(["docs-hero", "bleed", "deep"]);
    });

    it("should throw on a still that Studio did not record", async () => {
      await expect(
        render({
          still: "console/missing/hero",
          layout: "bleed",
          tone: "soft",
          palette: "teal-blue",
        }),
      ).rejects.toThrow(
        "no hero still console/missing/hero: add a hero entry to studio/videos.ts and run pnpm batch",
      );
    });

    it("should lift the toolbar of a layered still out of the window", async () => {
      const root = await render({
        still: "console/line-plots/hero",
        layout: "layered",
        tone: "deep",
        palette: "blue-violet",
      });
      const [toolbar] = withClass(root, "toolbar");
      expect(images(toolbar)).toEqual([
        CDN.url(`console/line-plots/hero/${linePlots.version}/window`, "webp", "light"),
      ]);
    });

    it("should throw on a layered still with no toolbar", async () => {
      await expect(
        render({
          still: "console/schematics/hero",
          layout: "layered",
          tone: "deep",
          palette: "violet-pink",
        }),
      ).rejects.toThrow(
        "the console/schematics/hero still records no toolbar for the layered layout",
      );
    });
  });

  describe("inset", () => {
    it("should mark the hero inset instead of a tone", async () => {
      const root = await render({
        still: "console/schematics/hero",
        layout: "framed",
        inset: true,
        palette: "green-blue",
      });
      expect(classes(root)).toEqual(["docs-hero", "framed", "inset"]);
    });

    it("should throw on a layered inset hero", async () => {
      await expect(
        render({
          still: "console/line-plots/hero",
          layout: "layered",
          inset: true,
          palette: "blue-violet",
        }),
      ).rejects.toThrow(
        "the console/line-plots/hero hero cannot be both layered and inset",
      );
    });
  });

  describe("media", () => {
    it("should show an image that Studio did not capture", async () => {
      const root = await render({
        image: "releases/0-49-0/thumbnail",
        extension: "webp",
        themed: false,
        layout: "framed",
        tone: "soft",
        palette: "teal-violet",
      });
      expect(images(root)).toEqual([CDN.url("releases/0-49-0/thumbnail", "webp")]);
      expect(classes(root)).toEqual(["docs-hero", "framed", "soft", "media"]);
    });

    it("should show a video that Studio did not capture", async () => {
      const root = await render({
        video: "releases/0-57-0/panels",
        layout: "framed",
        tone: "deep",
        palette: "blue-violet",
      });
      expect(withClass(root, "docs-video")).toHaveLength(1);
      expect(images(root)).toEqual([]);
    });
  });
});
