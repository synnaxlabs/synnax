// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { computed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it, onTestFinished } from "vitest";
import { render } from "vitest-browser-react";

import { Keys } from "@/components/keys/Keys";

const center = (el: Element): number => {
  const { top, bottom } = el.getBoundingClientRect();
  return (top + bottom) / 2;
};

const find = (container: Element, selector: string): Element => {
  const el = container.querySelector(selector);
  if (el == null) throw new Error(`${selector} not found`);
  return el;
};

describe("Keys", () => {
  it("should center its chip in a button", async () => {
    const { container } = await render(
      <Button.Button variant="outlined" size="medium">
        Search
        <Keys trigger={["Control", "K"]} />
      </Button.Button>,
    );
    const button = find(container, "button");
    const chip = find(container, ".docs-keys__mac .lyra-triggers__keys");
    expect(center(chip)).toBeCloseTo(center(button), 0);
  });

  it("should stay hidden under the hidden attribute", async () => {
    const { container } = await render(<Keys trigger={["Control", "K"]} hidden />);
    expect(computed(find(container, ".docs-keys"), "display")).toBe("none");
  });

  it("should let a class hide it from a stylesheet that loads first", async () => {
    const style = document.createElement("style");
    style.textContent = ".hide-keys { display: none; }";
    document.head.prepend(style);
    onTestFinished(() => style.remove());
    const { container } = await render(
      <Keys trigger={["Control", "K"]} className="hide-keys" />,
    );
    expect(computed(find(container, ".docs-keys"), "display")).toBe("none");
  });
});
