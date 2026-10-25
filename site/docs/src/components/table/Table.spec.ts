// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { type Element, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { beforeAll, describe, expect, it } from "vitest";

import Table from "@/components/table/Table.astro";

const SHORT_NAME = "a".repeat(24);
const LONG_NAME = "a".repeat(25);

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const text = (node: Element): string =>
  node.children
    .map((child) =>
      child.type === "text" ? child.value : child.type === "element" ? text(child) : "",
    )
    .join("");

const has = (node: Element, property: string): boolean =>
  node.properties[property] !== undefined;

const isNum = (node: Element): boolean => {
  const { className } = node.properties;
  return Array.isArray(className) && className.includes("num");
};

const mark = (cell: Element): unknown =>
  elements(cell).find((node) => node.tagName === "svg")?.properties.ariaLabel;

// The HTML that MDX passes to the table, which already escapes the text of each cell.
const slot = (header: string[], body: string[][]): string => {
  const line = (tag: string, cells: string[]): string =>
    `<tr>${cells.map((cell) => `<${tag}>${cell}</${tag}>`).join("")}</tr>`;
  const lines = body.map((cells) => line("td", cells)).join("");
  return `<thead>${line("th", header)}</thead><tbody>${lines}</tbody>`;
};

interface Rendered {
  html: string;
  shape: unknown;
  headers: Element[];
  /** The cells of each column, top to bottom. */
  columns: Element[][];
}

let container: AstroContainer;

beforeAll(async () => {
  container = await AstroContainer.create();
});

const render = async (header: string[], ...body: string[][]): Promise<Rendered> => {
  const html = await container.renderToString(Table, {
    slots: { default: slot(header, body) },
  });
  const nodes = elements(fromHtml(html, { fragment: true }));
  const cells = nodes.filter((node) => node.tagName === "td");
  return {
    html,
    shape: nodes[0].properties.dataShape,
    headers: nodes.filter((node) => node.tagName === "th"),
    columns: header.map((_, i) => cells.filter((_, j) => j % header.length === i)),
  };
};

describe("Table", () => {
  describe("labels", () => {
    it("should label each cell with the text of its column header", async () => {
      const { columns } = await render(
        ["Name", "Range <small><code>min_val</code></small>"],
        ["Voltage", "0 to 1"],
      );
      expect(columns.map(([cell]) => cell.properties.dataLabel)).toEqual([
        "Name",
        "Range min_val",
      ]);
    });

    it("should keep a header with &, <, or a quote escaped", async () => {
      const header = "A &amp; &lt;B&gt; &quot;C&quot;";
      const { html, columns } = await render(["Name", header], ["Voltage", "On"]);
      expect(html).toContain(`data-label="${header}"`);
      expect(columns[1][0].properties.dataLabel).toBe('A & <B> "C"');
    });
  });

  describe("number columns", () => {
    it("should mark a column of quantities and its header as numbers", async () => {
      const { headers, columns } = await render(
        ["Setting", "Value"],
        ["Sample rate", "10 Hz"],
        ["Latency", "&lt; 1 µs"],
        ["Buffer", "1,000"],
        ["Load", "50%"],
      );
      expect(headers.map(isNum)).toEqual([false, true]);
      expect(columns[1].map(isNum)).toEqual([true, true, true, true]);
    });

    it("should ignore an ellipsis or an empty cell in a number column", async () => {
      const { columns } = await render(
        ["Name", "Value"],
        ["a", "10"],
        ["b", "…"],
        ["c", "..."],
        ["d", ""],
      );
      expect(columns[1].map(isNum)).toEqual([true, true, true, true]);
    });

    it("should not mark a column that holds a word", async () => {
      const { headers, columns } = await render(
        ["Name", "Value"],
        ["a", "10"],
        ["b", "Fast"],
      );
      expect(headers.map(isNum)).toEqual([false, false]);
      expect(columns[1].map(isNum)).toEqual([false, false]);
    });

    it("should not mark a number in code", async () => {
      const { columns } = await render(["Name", "Value"], ["a", "<code>10</code>"]);
      expect(columns[1].map(isNum)).toEqual([false]);
    });

    it("should never mark the first column", async () => {
      const { headers, columns } = await render(
        ["Port", "Name"],
        ["1", "a"],
        ["2", "b"],
      );
      expect(headers.map(isNum)).toEqual([false, false]);
      expect(columns[0].map(isNum)).toEqual([false, false]);
    });
  });

  describe("marks", () => {
    it("should show Yes and No as marks", async () => {
      const { columns } = await render(
        ["Name", "Supported"],
        ["Voltage", "Yes"],
        ["Current", "No"],
      );
      expect(columns[1].map(mark)).toEqual(["Yes", "No"]);
      expect(columns[1].map(text)).toEqual(["", ""]);
    });

    it("should keep Yes and No as text in the first column", async () => {
      const { columns } = await render(["Name", "Value"], ["Yes", "a"], ["No", "b"]);
      expect(columns[0].map(mark)).toEqual([undefined, undefined]);
      expect(columns[0].map(text)).toEqual(["Yes", "No"]);
    });
  });

  describe("whole cells", () => {
    it("should keep a short name in the first column whole", async () => {
      const { columns } = await render(["Name", "Value"], [SHORT_NAME, "a"]);
      expect(has(columns[0][0], "dataWhole")).toBe(true);
    });

    it("should let a long name in the first column wrap", async () => {
      const { columns } = await render(["Name", "Value"], [LONG_NAME, "a"]);
      expect(has(columns[0][0], "dataWhole")).toBe(false);
    });

    it("should keep a quantity whole in any column", async () => {
      const { columns } = await render(
        ["Name", "Value"],
        ["a", "10 Hz"],
        ["b", "Fast"],
      );
      expect(columns[1].map((cell) => has(cell, "dataWhole"))).toEqual([true, false]);
    });
  });

  describe("cells", () => {
    it("should mark an empty cell", async () => {
      const { columns } = await render(["Name", "Value"], ["a", ""], ["b", "c"]);
      expect(columns[1].map((cell) => has(cell, "dataEmpty"))).toEqual([true, false]);
    });

    it("should wrap the content of each cell in a span", async () => {
      const { columns } = await render(["Name", "Value"], ["a", "<code>b</code>"]);
      const [span] = columns[1][0].children as Element[];
      expect(columns[1][0].children).toHaveLength(1);
      expect(span.tagName).toBe("span");
      expect(elements(span).map((node) => node.tagName)).toEqual(["code"]);
    });
  });

  describe("phone layout", () => {
    it("should keep a table of three short columns compact", async () => {
      const { shape } = await render(
        ["A", "B", "C"],
        [SHORT_NAME, SHORT_NAME, SHORT_NAME],
      );
      expect(shape).toBe("compact");
    });

    it("should stack a table of four columns", async () => {
      const { shape } = await render(["A", "B", "C", "D"], ["a", "b", "c", "d"]);
      expect(shape).toBe("stack");
    });

    it("should stack a table with a cell longer than a short name", async () => {
      const { shape } = await render(["A", "B", "C"], ["a", LONG_NAME, "c"]);
      expect(shape).toBe("stack");
    });
  });
});
