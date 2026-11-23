// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { stubImageSize } from "@synnaxlabs/site-common/testutil/image";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import type { ComponentProps } from "astro/types";
import type { Element, Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { describe, expect, it } from "vitest";

import { CDN } from "@/components/media/cdn";
import Image from "@/components/media/Image.astro";
import { IMAGES } from "@/util/images";

const render = async (props: ComponentProps<typeof Image>): Promise<Root> => {
  const container = await AstroContainer.create();
  return fromHtml(await container.renderToString(Image, { props }), { fragment: true });
};

const all = (node: Root | Element, tag: string): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element"
      ? [...(child.tagName === tag ? [child] : []), ...all(child, tag)]
      : [],
  );

const attr = (element: Element, name: string): unknown => element.properties[name];

// The CDN URLs that a srcset resizes.
const hrefs = (element: Element): string[] => [
  ...new Set(
    String(attr(element, "srcSet"))
      .split(",")
      .map(
        (entry) =>
          new URL(
            entry.trim().split(" ")[0],
            "https://docs.synnaxlabs.com",
          ).searchParams.get("url") ?? "",
      ),
  ),
];

describe("Image", () => {
  it("should offer the dark version by color scheme and fall back to light", async () => {
    stubImageSize(1600, 1000);
    const root = await render({ id: "shot", alt: "Shot" });
    expect(all(root, "source").map((s) => [attr(s, "media"), hrefs(s)])).toEqual([
      ["(prefers-color-scheme: dark)", [`${CDN.ROOT}/shot-dark.png`]],
      ["(prefers-color-scheme: dark)", [`${CDN.ROOT}/shot-dark.png`]],
      [undefined, [`${CDN.ROOT}/shot-light.png`]],
    ]);
    const [picture] = all(root, "picture");
    expect(attr(picture, "className")).toEqual(["docs-picture"]);
    const [img] = all(picture, "img");
    expect(hrefs(img)).toEqual([`${CDN.ROOT}/shot-light.png`]);
    expect(attr(img, "alt")).toBe("Shot");
    expect(attr(img, "className")).toEqual(["docs-image"]);
    expect(attr(img, "loading")).toBe("lazy");
  });

  it("should reserve the space of the image", async () => {
    stubImageSize(1600, 1000);
    const [img] = all(await render({ id: "shot", alt: "Shot" }), "img");
    expect(attr(img, "width")).toBe(1600);
    expect(attr(img, "height")).toBe(1000);
  });

  it("should request only the widths that Vercel resizes to", async () => {
    stubImageSize(1600, 1000);
    const root = await render({ id: "shot", alt: "Shot" });
    const widths = [...all(root, "source"), ...all(root, "img")].flatMap((element) =>
      String(attr(element, "srcSet"))
        .split(",")
        .map((entry) => {
          const [url, descriptor] = entry.trim().split(" ");
          const w = new URL(url, "https://docs.synnaxlabs.com").searchParams.get("w");
          expect(`${w}w`).toBe(descriptor);
          return Number(w);
        }),
    );
    expect(new Set(widths)).toEqual(new Set([640, 960, 1280]));
    for (const width of widths) expect(IMAGES.sizes).toContain(width);
  });

  it("should offer one version when not themed", async () => {
    stubImageSize(1600, 1000);
    const root = await render({ id: "shot", alt: "", themed: false });
    expect(all(root, "source").map(hrefs)).toEqual([[`${CDN.ROOT}/shot.png`]]);
  });

  it("should add a class to the docs image class", async () => {
    stubImageSize(1600, 1000);
    const root = await render({ id: "shot", alt: "Shot", class: "docs-tour__image" });
    const [img] = all(root, "img");
    expect(attr(img, "className")).toEqual(["docs-image", "docs-tour__image"]);
  });

  it("should render a vector image as is, at its own size", async () => {
    stubImageSize(800, 400);
    const root = await render({ id: "logo", alt: "", extension: "svg", themed: false });
    expect(all(root, "picture")).toEqual([]);
    const [img] = all(root, "img");
    expect(attr(img, "src")).toBe(`${CDN.ROOT}/logo.svg`);
    expect(attr(img, "width")).toBe(800);
    expect(attr(img, "height")).toBe(400);
  });

  it("should throw for a themed vector image", async () => {
    await expect(render({ id: "logo", alt: "", extension: "svg" })).rejects.toThrow(
      "docs image logo: a vector image cannot be themed",
    );
  });
});
