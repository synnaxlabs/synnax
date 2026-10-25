// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";

const setup = (target: number | null) =>
  renderHook(({ target }) => Primitive.useDraft(target), {
    initialProps: { target },
  });

const sendFive = (target: number | null) => {
  const h = setup(target);
  act(() => h.result.current.edit(5));
  act(() => h.result.current.send());
  return h;
};

describe("Primitive.useDraft", () => {
  it("should follow the target without an edit or a send", () => {
    const { result, rerender } = setup(3);
    expect(result.current).toMatchObject({ value: 3, dirty: false });
    rerender({ target: 4 });
    expect(result.current).toMatchObject({ value: 4, dirty: false });
  });

  it("should show an edit and mark it dirty only while it differs", () => {
    const { result } = setup(3);
    act(() => result.current.edit(5));
    expect(result.current).toMatchObject({ value: 5, dirty: true });
    act(() => result.current.edit(3));
    expect(result.current).toMatchObject({ value: 3, dirty: false });
  });

  it("should follow the target after an edit back to it", () => {
    const { result, rerender } = setup(3);
    act(() => result.current.edit(5));
    act(() => result.current.edit(3));
    rerender({ target: 4 });
    expect(result.current).toMatchObject({ value: 4, dirty: false });
  });

  it("should follow the target after it reaches an unsent edit", () => {
    const { result, rerender } = setup(3);
    act(() => result.current.edit(5));
    rerender({ target: 5 });
    rerender({ target: 6 });
    expect(result.current).toMatchObject({ value: 6, dirty: false });
  });

  it("should keep an edit that matches the target over a pending send", () => {
    const { result } = sendFive(3);
    act(() => result.current.edit(3));
    expect(result.current.value).toBe(3);
  });

  it("should keep a sent value until the target reports it", () => {
    const { result, rerender } = sendFive(3);
    expect(result.current).toMatchObject({ value: 5, dirty: true });
    rerender({ target: 3 });
    expect(result.current).toMatchObject({ value: 5, dirty: true });
    rerender({ target: 5 });
    expect(result.current).toMatchObject({ value: 5, dirty: false });
  });

  it("should show a reported value that differs from the one sent", () => {
    const { result, rerender } = sendFive(3);
    rerender({ target: 4.9999 });
    expect(result.current).toMatchObject({ value: 4.9999, dirty: false });
  });

  it("should keep following the target after a send settles", () => {
    const { result, rerender } = sendFive(3);
    rerender({ target: 5 });
    rerender({ target: 3 });
    expect(result.current).toMatchObject({ value: 3, dirty: false });
  });

  it("should show an edit over a pending send until it is discarded", () => {
    const { result } = sendFive(3);
    act(() => result.current.edit(7));
    expect(result.current.value).toBe(7);
    act(() => result.current.edit(null));
    expect(result.current).toMatchObject({ value: 5, dirty: true });
  });

  it("should settle at once when the sent value is the one the target holds", () => {
    const { result } = setup(3);
    act(() => result.current.send());
    expect(result.current).toMatchObject({ value: 3, dirty: false });
  });

  it("should hold nothing when it sends with no value known", () => {
    const { result, rerender } = setup(null);
    act(() => result.current.send());
    expect(result.current).toMatchObject({ value: null, dirty: false });
    rerender({ target: 4 });
    expect(result.current).toMatchObject({ value: 4, dirty: false });
  });
});
