// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { createRenderCounter } from "@/testutil/renders";

describe("createRenderCounter", () => {
  const Clickable = ({ label }: { label: string }): ReactElement => {
    const [count, setCount] = useState(0);
    return (
      <button onClick={() => setCount((c) => c + 1)}>
        {label} {count}
      </button>
    );
  };

  it("should not count the mount", () => {
    const { Counted, counts } = createRenderCounter();
    render(
      <Counted id="a">
        <Clickable label="a" />
      </Counted>,
    );
    expect(counts.size).toBe(0);
  });

  it("should count a re-render that starts inside its children", () => {
    const { Counted, counts } = createRenderCounter();
    const c = render(
      <Counted id="a">
        <Clickable label="a" />
      </Counted>,
    );
    fireEvent.click(c.getByText("a 0"));
    fireEvent.click(c.getByText("a 1"));
    expect(counts.get("a")).toBe(2);
  });

  it("should count each id apart from the others", () => {
    const { Counted, counts } = createRenderCounter();
    const c = render(
      <>
        <Counted id="a">
          <Clickable label="a" />
        </Counted>
        <Counted id="b">
          <Clickable label="b" />
        </Counted>
      </>,
    );
    fireEvent.click(c.getByText("a 0"));
    expect([...counts]).toEqual([["a", 1]]);
  });

  it("should not count a parent re-render that skips its element", () => {
    const { Counted, counts } = createRenderCounter();
    const counted = (
      <Counted id="a">
        <span>static</span>
      </Counted>
    );
    const Parent = (): ReactElement => {
      const [count, setCount] = useState(0);
      return (
        <>
          <button onClick={() => setCount((c) => c + 1)}>parent {count}</button>
          {counted}
        </>
      );
    };
    const c = render(<Parent />);
    fireEvent.click(c.getByText("parent 0"));
    expect(counts.size).toBe(0);
  });

  it("should clear the counts on reset", () => {
    const { Counted, counts, reset } = createRenderCounter();
    const c = render(
      <Counted id="a">
        <Clickable label="a" />
      </Counted>,
    );
    fireEvent.click(c.getByText("a 0"));
    reset();
    expect(counts.size).toBe(0);
    fireEvent.click(c.getByText("a 1"));
    expect(counts.get("a")).toBe(1);
  });
});
