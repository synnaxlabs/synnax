// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Results } from "@/components/search/Results";

const hit = (title: string) => ({
  objectID: title,
  title,
  href: `/reference/${title}`,
});

describe("Results", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("should keep the latest results when an older query answers last", async () => {
    class Observer {
      observe(): void {}
      unobserve(): void {}
      disconnect(): void {}
    }
    vi.stubGlobal("IntersectionObserver", Observer);
    vi.stubGlobal("ResizeObserver", Observer);
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
    const { container } = render(<Results />);
    const input = container.querySelector("input")!;
    fireEvent.change(input, { target: { value: "c" } });
    fireEvent.change(input, { target: { value: "channels" } });
    expect(pending).toHaveLength(2);
    await act(async () => pending[1](["newest"]));
    await act(async () => pending[0](["stale"]));
    const titles = [...container.querySelectorAll(".search-result__title")].map(
      (el) => el.textContent,
    );
    expect(titles).toEqual(["newest"]);
  });
});
