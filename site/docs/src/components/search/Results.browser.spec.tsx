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

const hit = (title: string) => ({
  objectID: title,
  title,
  href: `/reference/${title}`,
});

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
              resolve({ json: async () => ({ hits: titles.map(hit) }) }),
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
});
