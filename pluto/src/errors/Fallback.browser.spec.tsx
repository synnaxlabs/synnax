// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { computed, renderThemed, resolve } from "@synnaxlabs/lyra/styletest";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Errors } from "@/errors";

const WIDTH = 600;

// The Console turns off text selection for every element but inputs and labels
// (console/src/app/App.css).
const CONSOLE_RULES = "*:not(:is(input, label)) { user-select: none !important; }";

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
  let consoleRules: HTMLStyleElement;

  beforeEach(() => {
    consoleRules = document.createElement("style");
    consoleRules.textContent = CONSOLE_RULES;
    document.head.appendChild(consoleRules);
  });

  afterEach(() => consoleRules.remove());

  it("should let the user select the message inside the Console", async () => {
    const content = await renderFallback("Something broke");
    expect(computed(part(content, "message"), "user-select")).toBe("text");
    expect(computed(part(content, "stack"), "user-select")).toBe("text");
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
