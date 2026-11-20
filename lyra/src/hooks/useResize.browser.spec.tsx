// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { useResize, type UseResizeHandler } from "@/hooks";
import { renderThemed } from "@/styletest";

const BOX = { width: 100, height: 50 };

/** Swaps its box for a new one of the same size and place. */
const Swap = ({ onResize }: { onResize: UseResizeHandler }): ReactElement => {
  const [first, setFirst] = useState(true);
  const ref = useResize(onResize);
  return (
    <>
      <button onClick={() => setFirst((prev) => !prev)}>Swap</button>
      {first ? (
        <div key="first" ref={ref} style={BOX} />
      ) : (
        <div key="second" ref={ref} style={BOX} />
      )}
    </>
  );
};

describe("useResize", () => {
  it("should report the first box of an element that replaces another", async () => {
    const onResize = vi.fn();
    await renderThemed(<Swap onResize={onResize} />);
    await expect.poll(() => onResize.mock.calls.length).toBe(1);
    await userEvent.click(page.getByRole("button", { name: "Swap" }));
    await expect.poll(() => onResize.mock.calls.length).toBe(2);
  });
});
