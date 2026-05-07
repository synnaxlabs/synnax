// Copyright 2026 Synnax Labs, Inc.
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

import { List } from "@/list";
import { state } from "@/state";

describe("pager", () => {
  it("should fetch the correct page when onFetchMore is called", () => {
    let currParams: List.PagerParams | undefined;
    const { result } = renderHook(() => {
      const retrieve = useCallback(
        (setter: state.SetArg<List.PagerParams, Partial<List.PagerParams>>) => {
          const nextParams = state.executeSetter(setter, currParams ?? {});
          currParams = nextParams;
          return nextParams;
        },
        [],
      );
      return List.usePager({ retrieve });
    });
    result.current.fetchMore();
    expect(currParams).toEqual({
      offset: 0,
      limit: 10,
      searchTerm: "",
    });
    result.current.fetchMore();
    expect(currParams).toEqual({
      offset: 10,
      limit: 10,
      searchTerm: "",
    });
  });

  it("should reset the offset when onSearch is called", () => {
    let currParams: List.PagerParams | undefined;
    const { result } = renderHook(() => {
      const retrieve = useCallback(
        (setter: state.SetArg<List.PagerParams, Partial<List.PagerParams>>) => {
          const nextParams = state.executeSetter(setter, currParams ?? {});
          currParams = nextParams;
          return nextParams;
        },
        [],
      );
      return List.usePager({ retrieve });
    });
    result.current.search("test");
    expect(currParams).toEqual({
      offset: 0,
      limit: 10,
      searchTerm: "test",
    });
    result.current.fetchMore();
    expect(currParams).toEqual({
      offset: 10,
      limit: 10,
      searchTerm: "test",
    });
    result.current.search("test2");
    expect(currParams).toEqual({
      offset: 0,
      limit: 10,
      searchTerm: "test2",
    });
  });
});
