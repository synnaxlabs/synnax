// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { start } from "@/components/integrations/vendor";

const vendor = (): HTMLElement => document.querySelector(".integrations-vendor")!;

const cell = (name: string): Element =>
  document.querySelector(`.integration-cell[data-vendor="${name}"]`)!;

const render = (): void => {
  document.body.innerHTML = `
    <span class="integrations-vendor">existing</span>
    <div class="integration-cell" data-vendor="LabJack"
      style="--brand-color: #C1131E;"></div>
    <div class="integration-cell" data-vendor="Modbus"
      style="--brand-color: #6B4C9A;"></div>`;
  start();
};

describe("vendor", () => {
  it("should name the vendor under the pointer in its brand color", () => {
    render();
    cell("LabJack").dispatchEvent(new MouseEvent("mouseenter"));
    expect(vendor().textContent).toBe("LabJack");
    expect(vendor().style.color).toBe("rgb(193, 19, 30)");
  });

  it("should follow the pointer from one vendor to the next", () => {
    render();
    cell("LabJack").dispatchEvent(new MouseEvent("mouseenter"));
    cell("LabJack").dispatchEvent(new MouseEvent("mouseleave"));
    cell("Modbus").dispatchEvent(new MouseEvent("mouseenter"));
    expect(vendor().textContent).toBe("Modbus");
    expect(vendor().style.color).toBe("rgb(107, 76, 154)");
  });

  it("should restore the idle text and color when the pointer leaves", () => {
    render();
    cell("LabJack").dispatchEvent(new MouseEvent("mouseenter"));
    cell("LabJack").dispatchEvent(new MouseEvent("mouseleave"));
    expect(vendor().textContent).toBe("existing");
    expect(vendor().style.color).toBe("");
  });

  it("should throw on a cell that names no vendor", () => {
    document.body.innerHTML = `
      <span class="integrations-vendor">existing</span>
      <div class="integration-cell"></div>`;
    expect(start).toThrow("an .integration-cell has no data-vendor");
  });
});
