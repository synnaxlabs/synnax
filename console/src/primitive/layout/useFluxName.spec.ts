// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it, vi } from "vitest";

import { createUseFluxName } from "@/primitive/layout/useFluxName";

interface RetrieveParams {
  onChange: (name: string) => void;
  addStatusOnFailure: boolean;
}

const setup = (useEnabled?: (key: string) => boolean | undefined) => {
  const update = vi.fn();
  const retrieve = vi.fn();
  let lastRetrieveParams: RetrieveParams | undefined;
  const useRename = () => ({ update });
  const useRetrieve = (params: RetrieveParams) => {
    lastRetrieveParams = params;
    return { retrieve };
  };
  const useName = createUseFluxName(useRename, useRetrieve, useEnabled);
  const onChange = vi.fn();
  return {
    update,
    retrieve,
    useName,
    onChange,
    getRetrieveParams: () => lastRetrieveParams,
  };
};

describe("createUseFluxName", () => {
  it("invokes update with the layout key and new name on rename", () => {
    const { update, useName, onChange } = setup();
    const { result } = renderHook(() => useName("layout-1", onChange));
    act(() => {
      result.current.onRename?.("renamed");
    });
    expect(update).toHaveBeenCalledTimes(1);
    expect(update).toHaveBeenCalledWith({ key: "layout-1", name: "renamed" });
  });

  it("invokes retrieve with the layout key", () => {
    const { retrieve, useName, onChange } = setup();
    const { result } = renderHook(() => useName("layout-1", onChange));
    act(() => {
      result.current.retrieve();
    });
    expect(retrieve).toHaveBeenCalledTimes(1);
    expect(retrieve).toHaveBeenCalledWith({ key: "layout-1" });
  });

  it("wires onChange through to the underlying retrieve hook with status disabled", () => {
    const { useName, onChange, getRetrieveParams } = setup();
    renderHook(() => useName("layout-1", onChange));
    const params = getRetrieveParams();
    expect(params?.onChange).toBe(onChange);
    expect(params?.addStatusOnFailure).toBe(false);
  });

  describe("when useEnabled is omitted", () => {
    it("treats the layout as enabled by default", () => {
      const { update, retrieve, useName, onChange } = setup();
      const { result } = renderHook(() => useName("layout-1", onChange));
      act(() => result.current.onRename?.("n"));
      act(() => result.current.retrieve());
      expect(update).toHaveBeenCalledWith({ key: "layout-1", name: "n" });
      expect(retrieve).toHaveBeenCalledWith({ key: "layout-1" });
    });
  });

  describe("when useEnabled returns true", () => {
    it("forwards the layout key and invokes update + retrieve", () => {
      const useEnabled = vi.fn(() => true);
      const { update, retrieve, useName, onChange } = setup(useEnabled);
      const { result } = renderHook(() => useName("layout-1", onChange));
      act(() => result.current.onRename?.("n"));
      act(() => result.current.retrieve());
      expect(useEnabled).toHaveBeenCalledWith("layout-1");
      expect(update).toHaveBeenCalledWith({ key: "layout-1", name: "n" });
      expect(retrieve).toHaveBeenCalledWith({ key: "layout-1" });
    });
  });

  describe("when useEnabled returns false", () => {
    it("suppresses both update and retrieve", () => {
      const { update, retrieve, useName, onChange } = setup(() => false);
      const { result } = renderHook(() => useName("layout-1", onChange));
      act(() => result.current.onRename?.("n"));
      act(() => result.current.retrieve());
      expect(update).not.toHaveBeenCalled();
      expect(retrieve).not.toHaveBeenCalled();
    });
  });

  describe("when useEnabled returns undefined", () => {
    it("suppresses both update and retrieve (strict === true check)", () => {
      const { update, retrieve, useName, onChange } = setup(() => undefined);
      const { result } = renderHook(() => useName("layout-1", onChange));
      act(() => result.current.onRename?.("n"));
      act(() => result.current.retrieve());
      expect(update).not.toHaveBeenCalled();
      expect(retrieve).not.toHaveBeenCalled();
    });
  });

  it("returns stable retrieve and onRename callbacks across renders with the same key", () => {
    const { useName, onChange } = setup();
    const { result, rerender } = renderHook(({ key }) => useName(key, onChange), {
      initialProps: { key: "layout-1" },
    });
    const first = result.current;
    rerender({ key: "layout-1" });
    expect(result.current.retrieve).toBe(first.retrieve);
    expect(result.current.onRename).toBe(first.onRename);
  });

  it("returns new callbacks when the layout key changes", () => {
    const { useName, onChange } = setup();
    const { result, rerender } = renderHook(({ key }) => useName(key, onChange), {
      initialProps: { key: "layout-1" },
    });
    const first = result.current;
    rerender({ key: "layout-2" });
    expect(result.current.retrieve).not.toBe(first.retrieve);
    expect(result.current.onRename).not.toBe(first.onRename);
  });

  it("targets the new key after the layout key changes", () => {
    const { update, retrieve, useName, onChange } = setup();
    const { result, rerender } = renderHook(({ key }) => useName(key, onChange), {
      initialProps: { key: "layout-1" },
    });
    rerender({ key: "layout-2" });
    act(() => result.current.onRename?.("n"));
    act(() => result.current.retrieve());
    expect(update).toHaveBeenCalledWith({ key: "layout-2", name: "n" });
    expect(retrieve).toHaveBeenCalledWith({ key: "layout-2" });
  });
});
