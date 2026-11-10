// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { read } from "./source.ts";

describe("read", () => {
  let root: string;
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "stylelint-source-"));
  });
  afterEach(() => fs.rmSync(root, { recursive: true, force: true }));

  const write = (files: Record<string, string>): void => {
    for (const [file, text] of Object.entries(files)) {
      const full = path.join(root, file);
      fs.mkdirSync(path.dirname(full), { recursive: true });
      fs.writeFileSync(full, text);
    }
  };

  const owners = (cls: string): string[] => [...read(root).owners(cls)];

  describe("owners", () => {
    it("should model each BEM helper", () => {
      write({
        "lyra/src/button/Button.tsx": `
          import { CSS } from "@/css";
          CSS.B("btn");
          CSS.BE("btn", "icon");
          CSS.BM("btn", "outlined");
          CSS.BEM("btn", "icon", "large");
          CSS.M("small");
        `,
      });
      for (const cls of [
        "lyra-btn",
        "lyra-btn__icon",
        "lyra-btn--outlined",
        "lyra-btn__icon--large",
        "lyra--small",
      ])
        expect(owners(cls)).toEqual(["lyra"]);
    });

    it("should use the console prefix for the helpers the Console defines", () => {
      write({
        "console/src/nav/Nav.tsx": `
          import { CSS } from "@/platform/css";
          CSS.B("nav");
        `,
      });
      expect(owners("console-nav")).toEqual(["console"]);
      expect(owners("lyra-nav")).toEqual([]);
    });

    it("should use the pluto prefix for the helpers Pluto defines", () => {
      write({
        "pluto/src/mosaic/Mosaic.tsx": `
          import { CSS } from "@/css";
          CSS.B("mosaic");
        `,
      });
      expect(owners("pluto-mosaic")).toEqual(["pluto"]);
      expect(owners("lyra-mosaic")).toEqual([]);
    });

    it("should model the bare helpers of a file that calls newBEM", () => {
      write({
        "lyra/src/css/css.ts": `
          const PREFIX = "lyra";
          export const { M } = newBEM(PREFIX);
          export const visible = (v: boolean) => M(v ? "visible" : "hidden");
        `,
      });
      expect(owners("lyra--visible")).toEqual(["lyra"]);
      expect(owners("lyra--hidden")).toEqual(["lyra"]);
      expect(owners("lyra--shown")).toEqual([]);
    });

    it("should credit a package with the classes of the helpers it calls", () => {
      write({
        "lyra/src/css/css.ts": `
          const PREFIX = "lyra";
          export const { M } = newBEM(PREFIX);
          export const selected = (s: boolean) => s && M("selected");
          export const editable = (e: boolean) => e && M("editable");
        `,
        "pluto/src/tree/Tree.tsx": `
          import { CSS } from "@/css";
          CSS.selected(true);
        `,
      });
      expect(owners("lyra--selected")).toEqual(["lyra", "pluto"]);
      expect(owners("lyra--editable")).toEqual(["lyra"]);
    });

    it("should fill a dynamic part with a string literal from any package", () => {
      write({
        "lyra/src/button/Button.tsx": `CSS.BM("btn", variant);`,
        "console/src/Toolbar.tsx": `const variant = "filled";`,
      });
      expect(owners("lyra-btn--filled")).toEqual(["lyra"]);
      expect(owners("lyra-btn--filed")).toEqual([]);
    });

    it("should fill a dynamic part with a number", () => {
      write({ "lyra/src/flex/Box.tsx": `CSS.M("bg", background.toString());` });
      expect(owners("lyra--bg-10")).toEqual(["lyra"]);
    });

    it("should read class names written as literals", () => {
      write({ "console/index.html": `<div id="root" class="console-root"></div>` });
      expect(owners("console-root")).toEqual(["console"]);
    });

    it("should read a literal that ends in a separator as a family", () => {
      write({
        "lyra/src/theming/Provider.tsx": `const CLASS_PREFIX = "lyra-theme-";`,
      });
      expect(owners("lyra-theme-synnax-dark")).toEqual(["lyra"]);
    });

    it("should not count a selector that reads a class", () => {
      write({
        "site/docs/src/collapse.ts": `document.querySelectorAll(".pluto-text");`,
      });
      expect(owners("pluto-text")).toEqual([]);
    });

    it("should not count classes that a style block or a spec names", () => {
      write({
        "site/docs/src/Page.astro": `<div /><style>.lyra-btn { color: red; }</style>`,
        "lyra/src/button/Button.spec.tsx": `expect(el.className).toContain("lyra-btn");`,
      });
      expect(owners("lyra-btn")).toEqual([]);
    });
  });

  describe("defined", () => {
    it("should read declarations and registrations in stylesheets", () => {
      write({
        "lyra/src/theming/base.css": `:root { --pluto-gray-l8: #000; }`,
        "lyra/src/tabs/Tabs.css": `@property --pluto-tabs-fade-start { syntax: "*"; }`,
      });
      const source = read(root);
      expect(source.defined("--pluto-gray-l8")).toBe(true);
      expect(source.defined("--pluto-tabs-fade-start")).toBe(true);
      expect(source.defined("--pluto-gray-l9")).toBe(false);
    });

    it("should give the properties that code sets the pluto prefix", () => {
      write({
        "lyra/src/a.tsx": `
          import { CSS } from "@/css";
          const style = { [CSS.variable("pack-br")]: 1, "--pluto-x": 2 };
          const hue = { [\`--pluto-hue-\${name}\`]: 3 };
          el.style.setProperty("--pluto-y", "1px");
        `,
        "console/index.html": `<div style="--console-z: 1px"></div>`,
      });
      const source = read(root);
      expect(source.defined("--pluto-pack-br")).toBe(true);
      expect(source.defined("--pluto-x")).toBe(true);
      expect(source.defined("--pluto-hue-1")).toBe(true);
      expect(source.defined("--pluto-y")).toBe(true);
      expect(source.defined("--console-z")).toBe(true);
    });

    it("should not count a property that code only reads or names", () => {
      write({
        "lyra/src/a.tsx": `
          /** Sized by \`--pluto-size\`. */
          getComputedStyle(el).getPropertyValue("--pluto-gray-l0");
        `,
      });
      const source = read(root);
      expect(source.defined("--pluto-gray-l0")).toBe(false);
      expect(source.defined("--pluto-size")).toBe(false);
    });

    it("should read the keys that Astro defines through define:vars", () => {
      write({
        "site/docs/src/Inline.astro": `<style define:vars={{ accent, "bg-color": c }}></style>`,
        "site/docs/src/Named.astro": `---
const colors = { "type-color": a, default: b };
---
<style define:vars={colors}></style>`,
      });
      const source = read(root);
      for (const property of ["--accent", "--bg-color", "--type-color", "--default"])
        expect(source.defined(property)).toBe(true);
    });
  });

  describe("packageOf", () => {
    it("should find the package that holds a file", () => {
      const source = read(root);
      expect(source.packageOf(path.join(root, "site/docs/src/a.css"))).toBe(
        "site/docs",
      );
      expect(source.packageOf(path.join(root, "x/ts/src/a.css"))).toBeUndefined();
    });
  });
});
