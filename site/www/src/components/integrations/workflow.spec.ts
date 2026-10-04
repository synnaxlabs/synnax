// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it } from "vitest";

import { start } from "@/components/integrations/workflow";

const tab = (step: number): HTMLElement =>
  document.querySelector(`.workflow-tab[data-step="${step}"]`)!;

const active = (selector: string): string[] =>
  [...document.querySelectorAll(`${selector}--active`)].map(
    (el) => el.getAttribute("data-step") ?? el.getAttribute("data-panel") ?? "",
  );

describe("workflow", () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <button class="workflow-tab workflow-tab--active" data-step="0"></button>
      <button class="workflow-tab" data-step="1"></button>
      <button class="workflow-tab" data-step="2"></button>
      <div class="workflow-panel workflow-panel--active" data-panel="0"></div>
      <div class="workflow-panel" data-panel="1"></div>
      <div class="workflow-panel" data-panel="2"></div>`;
    start();
  });

  it("should activate the clicked tab and its panel alone", () => {
    tab(2).click();
    expect(active(".workflow-tab")).toEqual(["2"]);
    expect(active(".workflow-panel")).toEqual(["2"]);
  });

  it("should keep the active tab and panel when the active tab is clicked", () => {
    tab(0).click();
    expect(active(".workflow-tab")).toEqual(["0"]);
    expect(active(".workflow-panel")).toEqual(["0"]);
  });

  it("should switch back to an earlier tab", () => {
    tab(1).click();
    tab(0).click();
    expect(active(".workflow-tab")).toEqual(["0"]);
    expect(active(".workflow-panel")).toEqual(["0"]);
  });

  it("should throw on a tab without a panel", () => {
    document.body.innerHTML = '<button class="workflow-tab" data-step="3"></button>';
    expect(start).toThrow("workflow step 3 has no panel");
  });
});
