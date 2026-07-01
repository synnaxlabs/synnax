// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/primitive/modals";
import { closeOf, renderModalHook } from "@/primitive/modals/testutil";

const VoidContent: Modals.Content<Record<never, never>, void> = () => null;
const NumberContent: Modals.Content<Record<never, never>, number> = () => null;
const Labeled: Modals.Content<{ label: string }, void> = ({ label }) => (
  <span>{label}</span>
);

describe("create", () => {
  it("should push a modal onto the stack when the opener is called", () => {
    const useOpen = Modals.create(VoidContent);
    const { result } = renderModalHook(useOpen);
    expect(result.current.store.isAnyOpen()).toBe(false);
    act(() => result.current.hook());
    expect(result.current.store.getState()).toHaveLength(1);
  });

  it("should return immediately without a result", () => {
    const useOpen = Modals.create(VoidContent);
    const { result } = renderModalHook(useOpen);
    let returned: unknown = "sentinel";
    act(() => {
      returned = result.current.hook();
    });
    expect(returned).toBeUndefined();
  });

  it("should bind the provided params onto the content", () => {
    const useOpen = Modals.create(Labeled);
    const { result } = renderModalHook(useOpen);
    act(() => result.current.hook({ label: "hello" }));
    const { props } = result.current.store.getState()[0].render() as ReactElement<
      Modals.ContentProps<{ label: string }>
    >;
    expect(props.label).toEqual("hello");
  });

  it("should return a stable opener across re-renders", () => {
    const useOpen = Modals.create(VoidContent);
    const { result, rerender } = renderModalHook(useOpen);
    const first = result.current.hook;
    rerender();
    expect(result.current.hook).toBe(first);
  });
});

describe("createPrompt", () => {
  it("should resolve with the result the content passes to close", async () => {
    const usePrompt = Modals.createPrompt<number>(NumberContent);
    const { result } = renderModalHook(usePrompt);
    let promise: Promise<number | null> | undefined;
    act(() => {
      promise = result.current.hook();
    });
    act(() => closeOf(result.current.store)(42));
    await expect(promise).resolves.toBe(42);
  });

  it("should resolve null when the content closes without a result", async () => {
    const usePrompt = Modals.createPrompt<number>(NumberContent);
    const { result } = renderModalHook(usePrompt);
    let promise: Promise<number | null> | undefined;
    act(() => {
      promise = result.current.hook();
    });
    act(() => closeOf(result.current.store)());
    await expect(promise).resolves.toBeNull();
  });

  it("should resolve null when the modal is dismissed", async () => {
    const usePrompt = Modals.createPrompt<number>(NumberContent);
    const { result } = renderModalHook(usePrompt);
    let promise: Promise<number | null> | undefined;
    act(() => {
      promise = result.current.hook();
    });
    act(() => result.current.store.getState()[0].dismiss());
    await expect(promise).resolves.toBeNull();
  });

  it("should return a stable prompt across re-renders", () => {
    const usePrompt = Modals.createPrompt<number>(NumberContent);
    const { result, rerender } = renderModalHook(usePrompt);
    const first = result.current.hook;
    rerender();
    expect(result.current.hook).toBe(first);
  });
});
