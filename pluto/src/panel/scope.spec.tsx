// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/panel";

const PANEL_KEY = "11111111-1111-4111-8111-111111111111";
const TAB_KEY = "22222222-2222-4222-8222-222222222222";

const nested = ({ children }: PropsWithChildren): ReactElement => (
  <Panel.Scope.Provider value={PANEL_KEY}>
    <Panel.TabScope.Provider value={TAB_KEY}>{children}</Panel.TabScope.Provider>
  </Panel.Scope.Provider>
);

describe("Panel scope", () => {
  it("should resolve the panel key from the surrounding panel scope", () => {
    const { result } = renderHook(() => Panel.useKey(), { wrapper: nested });
    expect(result.current).toEqual(PANEL_KEY);
  });

  it("should resolve the tab key from the surrounding tab scope", () => {
    const { result } = renderHook(() => Panel.useTabKey(), { wrapper: nested });
    expect(result.current).toEqual(TAB_KEY);
  });

  it("should resolve each level independently when nested", () => {
    const { result } = renderHook(
      () => ({ key: Panel.useKey(), tabKey: Panel.useTabKey() }),
      { wrapper: nested },
    );
    expect(result.current).toEqual({ key: PANEL_KEY, tabKey: TAB_KEY });
  });

  it("should throw when no panel scope is present", () => {
    expect(() => renderHook(() => Panel.useKey())).toThrow(
      "Panel scope requires a key",
    );
  });

  it("should throw when no tab scope is present", () => {
    expect(() => renderHook(() => Panel.useTabKey())).toThrow(
      "PanelTab scope requires a key",
    );
  });

  it("should let an explicit override win over the scope", () => {
    const { result } = renderHook(() => Panel.useKey(TAB_KEY), { wrapper: nested });
    expect(result.current).toEqual(TAB_KEY);
  });
});
