// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { Code } from "@/code";

const ORIGINAL = "const pressure = 1;";
const MODIFIED = "const pressure = 2;";

type RenderProps = Omit<Code.DiffEditorProps, "original" | "modified" | "language">;

const renderDiff = async (props: RenderProps = {}): Promise<void> => {
  await renderThemed(
    <Code.Provider>
      <div style={{ width: 800, height: 400 }}>
        <Code.DiffEditor
          original={ORIGINAL}
          modified={MODIFIED}
          language="plaintext"
          {...props}
        />
      </div>
    </Code.Provider>,
  );
};

const rect = (locator: Locator): DOMRect => locator.element().getBoundingClientRect();

// Inline mode draws the original text above the modified text, in its column, once the
// diff is computed.
const stacked = (): boolean => {
  const modified = rect(page.getByText(MODIFIED));
  return page
    .getByText(ORIGINAL)
    .elements()
    .some((el) => {
      const original = el.getBoundingClientRect();
      return original.left === modified.left && original.bottom <= modified.top;
    });
};

describe("Code.DiffEditor", () => {
  // Monaco and its diff worker load once per page, which takes seconds.
  beforeAll(async () => {
    await renderDiff({ inline: true });
    await vi.waitFor(() => expect(stacked()).toBe(true), { timeout: 30_000 });
  }, 40_000);

  it("should show each text under its label, side by side", async () => {
    await renderDiff({ originalLabel: "Committed", modifiedLabel: "On the Core" });
    await expect.element(page.getByText(ORIGINAL)).toBeVisible();
    await expect.element(page.getByText(MODIFIED)).toBeVisible();
    const originalLabel = rect(page.getByText("Committed"));
    const modifiedLabel = rect(page.getByText("On the Core"));
    const original = rect(page.getByText(ORIGINAL));
    const modified = rect(page.getByText(MODIFIED));
    expect(original.top).toBe(modified.top);
    expect(originalLabel.bottom).toBeLessThanOrEqual(original.top);
    expect(modifiedLabel.bottom).toBeLessThanOrEqual(modified.top);
    expect(original.left).toBeGreaterThanOrEqual(originalLabel.left);
    expect(original.right).toBeLessThanOrEqual(modifiedLabel.left);
    expect(modified.left).toBeGreaterThanOrEqual(modifiedLabel.left);
  });

  it("should show the original text above the modified text when inline", async () => {
    await renderDiff({ inline: true });
    await expect.poll(stacked).toBe(true);
  });

  it.each([
    ["original", ORIGINAL],
    ["modified", MODIFIED],
  ])("should not let a user edit the %s text", async (_, text) => {
    await renderDiff();
    await userEvent.click(page.getByText(text));
    await userEvent.keyboard("xyz");
    // Monaco draws an edit on the next frame.
    await new Promise((resolve) => requestAnimationFrame(resolve));
    await expect.element(page.getByText(text)).toBeVisible();
  });
});
