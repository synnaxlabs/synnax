// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  computed,
  renderThemed,
  resolve,
  selectedByTripleClick,
} from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Errors } from "@/errors";

const WIDTH = 600;

const renderFallback = async (message: string): Promise<Element> => {
  const error = new Error(message);
  error.name = "TestError";
  const { container } = await renderThemed(
    <div style={{ width: WIDTH, height: 800 }}>
      <Errors.Fallback error={error} resetErrorBoundary={() => {}} />
    </div>,
  );
  const content = container.querySelector(`.${CSS.BE("error-fallback", "content")}`);
  if (content == null) throw new Error("fallback content not found");
  return content;
};

const part = (content: Element, element: string): Element => {
  const el = content.querySelector(`.${CSS.BE("error-fallback", element)}`);
  if (el == null) throw new Error(`fallback ${element} not found`);
  return el;
};

describe("Errors.Fallback", () => {
  it("should let the user select the message and stack inside the Console", async () => {
    const content = await renderFallback("Something broke");
    expect(await selectedByTripleClick(part(content, "message"))).toContain(
      "Something broke",
    );
    expect(await selectedByTripleClick(part(content, "stack"))).not.toBe("");
  });

  it("should border its content in the error color", async () => {
    const content = await renderFallback("Something broke");
    expect(computed(content, "border-top-color")).toBe(
      resolve("var(--pluto-error-z-40)"),
    );
  });

  it("should wrap a long message inside its content", async () => {
    const content = await renderFallback("x".repeat(2000));
    const message = part(content, "message");
    expect(message.scrollWidth).toBeLessThanOrEqual(message.clientWidth);
    expect(content.getBoundingClientRect().right).toBeLessThanOrEqual(WIDTH);
  });
});
