// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Search } from "@/components/search";

const hit = (title: string, href = `/reference/${title}`) => ({
  objectID: title,
  title,
  href,
});

const openWith = async (hits: ReturnType<typeof hit>[]): Promise<void> => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => ({ json: async () => ({ hits }) })),
  );
  render(<Search.Search />);
  fireEvent.click(screen.getByRole("button", { name: /Search/ }));
  const input = document.querySelector<HTMLInputElement>(
    ".search-results__content input",
  )!;
  await act(async () => fireEvent.change(input, { target: { value: "read" } }));
};

const iconColors = (): string[] =>
  [...document.querySelectorAll<SVGElement>(".search-result__icon svg")].map(
    (el) => el.style.color,
  );

const prefersDark = (matches: boolean) =>
  vi.stubGlobal(
    "matchMedia",
    vi.fn(() => ({ matches, addEventListener: vi.fn(), removeEventListener: vi.fn() })),
  );

describe("Results", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("should keep the latest results when an older query answers last", async () => {
    const pending: ((titles: string[]) => void)[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(
        () =>
          new Promise((resolve) =>
            pending.push((titles) =>
              resolve({
                json: async () => ({ hits: titles.map((title) => hit(title)) }),
              }),
            ),
          ),
      ),
    );
    render(<Search.Search />);
    fireEvent.click(screen.getByRole("button", { name: /Search/ }));
    const input = document.querySelector<HTMLInputElement>(
      ".search-results__content input",
    )!;
    fireEvent.change(input, { target: { value: "c" } });
    fireEvent.change(input, { target: { value: "channels" } });
    expect(pending).toHaveLength(2);
    await act(async () => pending[1](["newest"]));
    await act(async () => pending[0](["stale"]));
    const titles = [...document.querySelectorAll(".search-result__title")].map(
      (el) => el.textContent,
    );
    expect(titles).toEqual(["newest"]);
  });

  describe("logos", () => {
    const HITS = [
      hit("LabJack read task", "/reference/hardware/labjack/read-task"),
      hit("channels"),
    ];

    it("should draw a vendor logo in its light brand color", async () => {
      prefersDark(false);
      await openWith(HITS);
      expect(iconColors()).toEqual(["rgb(141, 2, 0)", ""]);
    });

    it("should draw a vendor logo in its dark brand color", async () => {
      prefersDark(true);
      await openWith(HITS);
      expect(iconColors()).toEqual(["rgb(226, 96, 80)", ""]);
    });
  });
});
