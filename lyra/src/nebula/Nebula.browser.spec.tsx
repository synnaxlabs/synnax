// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Nebula } from "@/nebula";
import { Theming } from "@/theming";

const renderNebula = (): HTMLElement =>
  render(
    <Theming.Provider>
      <div style={{ position: "relative", width: 300, height: 200 }}>
        <Nebula.Nebula />
      </div>
    </Theming.Provider>,
  ).container;

const nextFrames = async (count: number): Promise<void> => {
  for (let i = 0; i < count; i++)
    await new Promise((resolve) => requestAnimationFrame(resolve));
};

describe("Nebula", () => {
  const errors: string[] = [];
  const handleError = (e: ErrorEvent): void => {
    errors.push(e.message);
  };
  window.addEventListener("error", handleError);
  afterEach(() => {
    errors.length = 0;
  });

  it("should draw nothing after its canvas leaves the page", async () => {
    const container = renderNebula();
    await nextFrames(2);
    container.remove();
    await nextFrames(3);
    expect(errors).toEqual([]);
  });

  it("should draw again when its canvas comes back", async () => {
    const getContext = vi.spyOn(HTMLCanvasElement.prototype, "getContext");
    const container = renderNebula();
    await nextFrames(2);
    container.remove();
    await nextFrames(2);
    getContext.mockClear();
    document.body.appendChild(container);
    await nextFrames(2);
    expect(getContext).toHaveBeenCalled();
    getContext.mockRestore();
  });
});
