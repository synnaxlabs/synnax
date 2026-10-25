// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { tokens } from "@synnaxlabs/arc";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { type Element, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { beforeAll, describe, expect, it } from "vitest";

import Item from "@/components/field/Item.astro";

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const text = (node: Element): string =>
  node.children
    .map((child) =>
      child.type === "text" ? child.value : child.type === "element" ? text(child) : "",
    )
    .join("")
    .replace(/\s+/g, " ")
    .trim();

const hasClass = (node: Element, name: string): boolean => {
  const { className } = node.properties;
  return Array.isArray(className) && className.includes(name);
};

interface Rendered {
  /** The first element with the class. Throws when there is none. */
  get: (name: string) => Element;
  has: (name: string) => boolean;
  /** The text of each part of the metadata line. */
  meta: () => string[];
}

let container: AstroContainer;

beforeAll(async () => {
  container = await AstroContainer.create();
});

const render = async (
  props: Record<string, unknown>,
  slots: Record<string, string> = {},
): Promise<Rendered> => {
  const html = await container.renderToString(Item, {
    props,
    slots: { default: "<p>Sets the address.</p>", ...slots },
  });
  const nodes = elements(fromHtml(html, { fragment: true }));
  const has = (name: string): boolean => nodes.some((node) => hasClass(node, name));
  const get = (name: string): Element => {
    const found = nodes.find((node) => hasClass(node, name));
    if (found == null) throw new Error(`no element has the class ${name}`);
    return found;
  };
  const meta = (): string[] => {
    const line = get("field__meta");
    return line.children
      .filter(
        (child): child is Element =>
          child.type === "element" && child.tagName === "span",
      )
      .map(text);
  };
  return { get, has, meta };
};

describe("Field.Item", () => {
  describe("names and anchors", () => {
    it("should anchor a flag at its name without the dashes", async () => {
      const { get } = await render({ name: "--listen", short: "-l" });
      expect(get("field").properties.id).toEqual("listen");
      const name = get("field__name");
      expect(name.properties.href).toEqual("#listen");
      expect(text(name)).toEqual("--listen, -l");
    });

    it("should anchor a label in kebab case", async () => {
      const { get } = await render({ label: "Data rate" });
      expect(get("field").properties.id).toEqual("data-rate");
      expect(text(get("field__label"))).toEqual("Data rate");
    });

    it("should anchor a name slot at its id", async () => {
      const { get } = await render({ id: "rate" }, { name: "<code>rate</code>" });
      expect(get("field").properties.id).toEqual("rate");
      expect(text(get("field__name"))).toEqual("rate");
    });

    it("should reject an entry with no name", async () => {
      await expect(render({})).rejects.toThrow(
        "Field.Item needs exactly one of name, label, or a name slot",
      );
    });

    it("should reject an entry with both a name and a label", async () => {
      await expect(render({ name: "rate", label: "Rate" })).rejects.toThrow(
        "Field.Item needs exactly one of name, label, or a name slot",
      );
    });

    it("should reject a name slot without an id", async () => {
      await expect(render({}, { name: "<code>rate</code>" })).rejects.toThrow(
        "Field.Item with a name slot needs an id",
      );
    });
  });

  describe("type", () => {
    it("should show the value of a flag in angle brackets", async () => {
      const { get } = await render({ name: "--listen", type: "address" });
      expect(text(get("field__type"))).toEqual("<address>");
    });

    it("should show the type of a field as it is", async () => {
      const { get } = await render({ name: "rate", type: "number" });
      expect(text(get("field__type"))).toEqual("number");
    });

    it("should link the type to the page that describes it", async () => {
      const { get } = await render({
        name: "rate",
        type: "Rate",
        typeHref: "/reference/client/time-types",
      });
      expect(get("field__type-link").properties.href).toEqual(
        "/reference/client/time-types",
      );
    });

    it("should show the default after the type", async () => {
      const { get } = await render({
        name: "--listen",
        type: "address",
        default: '"localhost:9090"',
      });
      expect(text(get("field__signature"))).toEqual('<address> = "localhost:9090"');
      expect(text(get("field__default"))).toEqual('"localhost:9090"');
    });

    it("should color the type and default as Arc code does", async () => {
      const { get } = await render({ name: "rate", type: "number", default: "0" });
      const { style } = get("field").properties;
      expect(style).toContain(
        `--type-color: light-dark(${tokens.type.light}, ${tokens.type.dark})`,
      );
      expect(style).toContain(
        `--default-color: light-dark(${tokens.string.light}, ${tokens.string.dark})`,
      );
    });

    it("should show a default without a type", async () => {
      const { get, has } = await render({ name: "--cluster-key", default: "0" });
      expect(text(get("field__signature"))).toEqual("= 0");
      expect(has("field__type")).toBe(false);
    });

    it("should leave out the signature when there is no type or default", async () => {
      const { has } = await render({ name: "--verbose" });
      expect(has("field__signature")).toBe(false);
    });

    it("should mark a required entry with an asterisk that readers skip", async () => {
      const { get } = await render({ name: "rate", required: true });
      expect(text(get("field__required"))).toEqual("* Required");
      expect(get("field__required-mark").properties.ariaHidden).toEqual("true");
    });
  });

  describe("metadata", () => {
    it("should list the client fields", async () => {
      const { meta } = await render({ name: "rate", field: ["rate", "data_rate"] });
      expect(meta()).toEqual(["Field rate, data_rate"]);
    });

    it("should name the Python and TypeScript fields when they differ", async () => {
      const { meta } = await render({
        name: "rate",
        field: { py: "data_rate", ts: "dataRate" },
      });
      expect(meta()).toEqual(["Python data_rate", "TypeScript dataRate"]);
    });

    it("should list the Console label, variable, and change rule", async () => {
      const { meta } = await render({
        name: "--listen",
        console: "Address",
        default: "localhost:9090",
        env: "SYNNAX_LISTEN",
        changeable: false,
      });
      expect(meta()).toEqual(["Console Address", "Env SYNNAX_LISTEN", "Cannot change"]);
    });

    it("should say when the value can change", async () => {
      const { meta } = await render({ name: "rate", changeable: true });
      expect(meta()).toEqual(["Can change"]);
    });

    it("should leave out the metadata line when there is none", async () => {
      const { has } = await render({ name: "rate" });
      expect(has("field__meta")).toBe(false);
    });
  });
});
