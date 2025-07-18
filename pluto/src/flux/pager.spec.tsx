// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { useCallback } from "react";
import { describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { state } from "@/state";

describe("pager", () => {
  it("should fetch the correct page when onFetchMore is called", () => {
    let currParams: Flux.PagerParams | undefined;
    const { result } = renderHook(() => {
      const retrieve = useCallback(
        (setter: state.SetArg<Flux.PagerParams, Partial<Flux.PagerParams>>) => {
          const nextParams = state.executeSetter(setter, currParams ?? {});
          currParams = nextParams;
          return nextParams;
        },
        [],
      );
      return Flux.usePager({ retrieve });
    });
    result.current.onFetchMore();
    expect(currParams).toEqual({
      offset: 0,
      limit: 10,
      term: undefined,
    });
    result.current.onFetchMore();
    expect(currParams).toEqual({
      offset: 10,
      limit: 10,
      term: undefined,
    });
  });

  it("should reset the offset when onSearch is called", () => {
    let currParams: Flux.PagerParams | undefined;
    const { result } = renderHook(() => {
      const retrieve = useCallback(
        (setter: state.SetArg<Flux.PagerParams, Partial<Flux.PagerParams>>) => {
          const nextParams = state.executeSetter(setter, currParams ?? {});
          currParams = nextParams;
          return nextParams;
        },
        [],
      );
      return Flux.usePager({ retrieve });
    });
    result.current.onSearch("test");
    expect(currParams).toEqual({
      offset: 0,
      limit: 10,
      term: "test",
    });
    result.current.onFetchMore();
    expect(currParams).toEqual({
      offset: 10,
      limit: 10,
      term: "test",
    });
    result.current.onSearch("test2");
    expect(currParams).toEqual({
      offset: 0,
      limit: 10,
      term: "test2",
    });
  });
});
