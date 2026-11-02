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
import { type Element, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { beforeAll, describe, expect, it } from "vitest";

import Item from "@/components/problem/Item.astro";

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
    .trim();

const hasClass = (node: Element, name: string): boolean => {
  const { className } = node.properties;
  return Array.isArray(className) && className.includes(name);
};

interface Rendered {
  /** Every element with the class. */
  all: (name: string) => Element[];
  /** The first element with the class. Throws when there is none. */
  get: (name: string) => Element;
}

let container: AstroContainer;

beforeAll(async () => {
  container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
});

const render = async (props: Record<string, unknown>): Promise<Rendered> => {
  const html = await container.renderToString(Item, {
    props,
    slots: { default: "<p>Restart the Driver.</p>" },
  });
  const nodes = elements(fromHtml(html, { fragment: true }));
  const all = (name: string): Element[] => nodes.filter((node) => hasClass(node, name));
  const get = (name: string): Element => {
    const [found] = all(name);
    if (found == null) throw new Error(`no element has the class ${name}`);
    return found;
  };
  return { all, get };
};

describe("Problem.Item", () => {
  it("should title an entry with its message and anchor it there", async () => {
    const { get } = await render({ message: "Failed to start task" });
    expect(get("problem").properties.id).toEqual("failed-to-start-task");
    const title = get("problem__title");
    expect(text(title)).toEqual("Failed to start task");
    expect(title.properties.href).toEqual("#failed-to-start-task");
    expect(hasClass(title, "problem__title--message")).toBe(true);
  });

  it("should title each message of one cause at one anchor", async () => {
    const { all, get } = await render({
      message: ["Driver not running", "Failed to reach the Driver"],
    });
    expect(get("problem").properties.id).toEqual("driver-not-running");
    expect(all("problem__title").map(text)).toEqual([
      "Driver not running",
      "Failed to reach the Driver",
    ]);
    expect(all("problem__title").map((t) => t.properties.href)).toEqual([
      "#driver-not-running",
      "#driver-not-running",
    ]);
  });

  it("should set the code in a symptom as inline code", async () => {
    const { get } = await render({ symptom: "The `Start` button stays gray" });
    const title = get("problem__title");
    expect(hasClass(title, "problem__title--message")).toBe(false);
    expect(text(title)).toEqual("The Start button stays gray");
    expect(elements(title).some((node) => node.tagName === "code")).toBe(true);
  });

  it("should anchor at the diagnostic code and show the severity", async () => {
    const { get } = await render({
      message: "Unknown channel",
      code: "ARC3001",
      severity: "error",
    });
    expect(get("problem").properties.id).toEqual("arc3001");
    expect(text(get("problem__code"))).toEqual("ARC3001");
    const severity = get("problem__severity");
    expect(text(severity)).toEqual("Error");
    expect(hasClass(severity, "problem__severity--error")).toBe(true);
  });

  it("should leave out the meta line when there is no code or severity", async () => {
    const { all } = await render({ message: "Failed to start task" });
    expect(all("problem__meta")).toEqual([]);
  });

  it("should cut a long anchor to 64 characters", async () => {
    const { get } = await render({ message: "a ".repeat(50) });
    expect(get("problem").properties.id).toEqual("a-".repeat(32));
  });

  it("should reject an entry with both a message and a symptom", async () => {
    await expect(
      render({ message: "Failed", symptom: "The task stops" }),
    ).rejects.toThrow("Problem.Item needs exactly one of message or symptom");
  });

  it("should reject an entry with neither a message nor a symptom", async () => {
    await expect(render({})).rejects.toThrow(
      "Problem.Item needs exactly one of message or symptom",
    );
  });
});
