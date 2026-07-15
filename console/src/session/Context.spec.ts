// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook, waitFor } from "@testing-library/react";
import { useSelector } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Session } from "@/session";

const useProbe = () => ({
  dispatch: Session.useDispatch(),
  selected: useSelector(Session.Cluster.selectSelectedKey),
  modals: Session.Modals.useStore("context-spec"),
});

describe("Session.Context", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("provides a live redux store and modals context to children", async () => {
    const { result } = renderHook(useProbe, { wrapper: Session.Context });

    await waitFor(() => {
      if (result.current == null) throw new Error("store has not resolved yet");
    });
    expect(result.current.modals).toBeDefined();
    expect(result.current.selected).toBeUndefined();

    act(() => {
      result.current.dispatch(Session.Cluster.select("DEMO"));
    });
    expect(result.current.selected).toBe("DEMO");
  });
});
