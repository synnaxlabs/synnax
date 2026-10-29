// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { computed, renderThemed, resolve } from "@/styletest";
import { Tag } from "@/tag";

const renderTag = async (label: string): Promise<Element> =>
  (await renderThemed(<Tag.Tag>{label}</Tag.Tag>)).container.querySelector(
    `.${CSS.B("tag")}`,
  )!;

describe("Tag", () => {
  it("should lift a bordered chip off the page", async () => {
    const tag = await renderTag("Pressure");
    expect(computed(tag, "background-color")).toBe(resolve("var(--pluto-gray-l0)"));
    expect(computed(tag, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    expect(computed(tag, "box-shadow")).toBe(
      resolve("var(--pluto-shadow-lift)", "box-shadow"),
    );
    expect(computed(tag, "border-top-left-radius")).toBe(
      resolve("var(--pluto-border-radius-small)", "border-top-left-radius"),
    );
  });

  it("should cut a long label at its maximum width", async () => {
    const tag = await renderTag("A very long tag label ".repeat(10));
    expect(tag.getBoundingClientRect().width).toBe(
      parseFloat(computed(tag, "max-width")),
    );
    const label = tag.lastElementChild!;
    expect(label.scrollWidth).toBeGreaterThan(label.clientWidth);
  });
});
