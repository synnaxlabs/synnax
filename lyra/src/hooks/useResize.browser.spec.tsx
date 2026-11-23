// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";
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

/** Moves a debounced ref from one box to another box of the same size. */
const Move = ({ onResize }: { onResize: UseResizeHandler }): ReactElement => {
  const [first, setFirst] = useState(true);
  const ref = useResize(onResize, { debounce: TimeSpan.seconds(1) });
  return (
    <>
      <button onClick={() => setFirst(false)}>Move</button>
      <div ref={first ? ref : undefined} style={BOX} />
      <div id="second" ref={first ? undefined : ref} style={BOX} />
    </>
  );
};

/**
 * Renders again inside the callback of a deeper observer, with a new handler for the
 * element of useResize above it.
 */
const Nested = ({
  onResize,
}: {
  onResize: (renders: number) => void;
}): ReactElement => {
  const [width, setWidth] = useState(10);
  const [renders, setRenders] = useState(0);
  const ref = useResize(() => onResize(renders));
  const deepRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const deep = deepRef.current;
    if (deep == null) return;
    const observer = new ResizeObserver(() =>
      flushSync(() => setRenders((r) => r + 1)),
    );
    observer.observe(deep);
    return () => observer.disconnect();
  }, []);
  return (
    <div>
      <button onClick={() => setWidth((w) => w + 10)}>Grow</button>
      <div ref={ref} style={BOX} />
      <div>
        <div>
          <div ref={deepRef} style={{ width, height: 10 }} />
        </div>
      </div>
    </div>
  );
};

/**
 * Grows its box on demand and toggles whether useResize observes it. The box keeps the
 * ref callback of its first render, so only the effect of useResize can observe again.
 */
const Toggle = ({
  onResize,
  enabled: initial,
}: {
  onResize: UseResizeHandler;
  enabled: boolean;
}): ReactElement => {
  const [enabled, setEnabled] = useState(initial);
  const [width, setWidth] = useState(BOX.width);
  const ref = useResize(onResize, { enabled });
  const [stableRef] = useState(() => ref);
  return (
    <>
      <button onClick={() => setEnabled((prev) => !prev)}>Toggle</button>
      <button onClick={() => setWidth((w) => w + 10)}>Grow</button>
      <div ref={stableRef} style={{ ...BOX, width }} />
    </>
  );
};

const nextFrames = async (): Promise<void> => {
  await new Promise((resolve) => requestAnimationFrame(resolve));
  await new Promise((resolve) => requestAnimationFrame(resolve));
};

describe("useResize", () => {
  it("should report the first box of an element that replaces another", async () => {
    const onResize = vi.fn();
    await renderThemed(<Swap onResize={onResize} />);
    await expect.poll(() => onResize.mock.calls.length).toBe(1);
    await userEvent.click(page.getByRole("button", { name: "Swap" }));
    await expect.poll(() => onResize.mock.calls.length).toBe(2);
  });

  it("should drop a pending report when the element changes", async () => {
    const onResize = vi.fn();
    await renderThemed(<Move onResize={onResize} />);
    await userEvent.click(page.getByRole("button", { name: "Move" }));
    expect(onResize).not.toHaveBeenCalled();
    await expect.poll(() => onResize.mock.calls.length, { timeout: 3000 }).toBe(1);
    await new Promise((resolve) => setTimeout(resolve, 1200));
    expect(onResize).toHaveBeenCalledTimes(1);
    expect(onResize.mock.calls[0][1]).toBe(document.getElementById("second"));
  });

  it("should not observe again when a deeper callback renders", async () => {
    const errors: string[] = [];
    const onError = (e: ErrorEvent): void => {
      errors.push(e.message);
    };
    window.addEventListener("error", onError);
    try {
      const onResize = vi.fn();
      await renderThemed(<Nested onResize={onResize} />);
      await expect.poll(() => onResize.mock.calls.length).toBe(1);
      await userEvent.click(page.getByRole("button", { name: "Grow" }));
      await new Promise((resolve) => requestAnimationFrame(resolve));
      await new Promise((resolve) => requestAnimationFrame(resolve));
      expect(errors).toEqual([]);
    } finally {
      window.removeEventListener("error", onError);
    }
  });

  it("should not call the handler when it is disabled", async () => {
    const onResize = vi.fn();
    await renderThemed(<Toggle onResize={onResize} enabled={false} />);
    await userEvent.click(page.getByRole("button", { name: "Grow" }));
    await nextFrames();
    expect(onResize).not.toHaveBeenCalled();
  });

  it("should stop calling the handler after it is disabled", async () => {
    const onResize = vi.fn();
    await renderThemed(<Toggle onResize={onResize} enabled />);
    await expect.poll(() => onResize.mock.calls.length).toBe(1);
    await userEvent.click(page.getByRole("button", { name: "Toggle" }));
    await userEvent.click(page.getByRole("button", { name: "Grow" }));
    await nextFrames();
    expect(onResize).toHaveBeenCalledTimes(1);
  });

  it("should call the handler after it is enabled", async () => {
    const onResize = vi.fn();
    await renderThemed(<Toggle onResize={onResize} enabled={false} />);
    await userEvent.click(page.getByRole("button", { name: "Toggle" }));
    await expect.poll(() => onResize.mock.calls.length).toBe(1);
    await userEvent.click(page.getByRole("button", { name: "Grow" }));
    await expect.poll(() => onResize.mock.calls.length).toBe(2);
  });

  it("should report a box that grew while it was disabled", async () => {
    const onResize = vi.fn();
    await renderThemed(<Toggle onResize={onResize} enabled />);
    await expect.poll(() => onResize.mock.calls.length).toBe(1);
    await userEvent.click(page.getByRole("button", { name: "Toggle" }));
    await userEvent.click(page.getByRole("button", { name: "Grow" }));
    await userEvent.click(page.getByRole("button", { name: "Toggle" }));
    await expect.poll(() => onResize.mock.calls.length).toBe(2);
    expect(box.width(onResize.mock.calls[1][0])).toBe(BOX.width + 10);
  });
});
