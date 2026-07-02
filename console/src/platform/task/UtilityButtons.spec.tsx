// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

const clickIcon = (container: HTMLElement, icon: string): void => {
  const button = container
    .querySelector(`[aria-label="pluto-icon--${icon}"]`)
    ?.closest("button");
  if (button == null) throw new Error(`button for icon ${icon} not found`);
  fireEvent.click(button);
};

describe("UtilityButtons", () => {
  it("should only show the JSON config button when the task has no key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: undefined, name: "T", config: {} },
    });
    expect(container.querySelector('[aria-label="pluto-icon--json"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--typescript"]')).toBeNull();
    expect(container.querySelector('[aria-label="pluto-icon--python"]')).toBeNull();
    expect(container.querySelector('[aria-label="pluto-icon--link"]')).toBeNull();
    expect(container.querySelector('[aria-label="pluto-icon--export"]')).toBeNull();
  });

  it("should show the code, link, and export buttons when the task has a key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task-123", name: "T", config: {} },
    });
    expect(container.querySelector('[aria-label="pluto-icon--json"]')).toBeTruthy();
    expect(
      container.querySelector('[aria-label="pluto-icon--typescript"]'),
    ).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--python"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--link"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--export"]')).toBeTruthy();
  });

  it("should invoke the clipboard payload builders without throwing when clicked", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task-123", name: "My Task", config: { foo: 1 } },
    });
    // Each copy button resolves its payload getter (TypeScript / Python / JSON code)
    // when clicked; exercising them covers the getter branches.
    clickIcon(container, "typescript");
    clickIcon(container, "python");
    clickIcon(container, "json");
    expect(container.querySelector('[aria-label="pluto-icon--json"]')).toBeTruthy();
  });
});
