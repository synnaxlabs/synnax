// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Theming } from "@/theming";

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Theming.Provider>{children}</Theming.Provider>
);

describe("Theming", () => {
  it("should provide the default theme", () => {
    const { result } = renderHook(() => Theming.useContext(), { wrapper });
    expect(result.current.theme.name).toEqual("Synnax Dark");
  });
  it("should toggle the theme", () => {
    const { result } = renderHook(() => Theming.useContext(), { wrapper });
    expect(result.current.theme.name).toEqual("Synnax Dark");
    act(() => result.current.toggleTheme());
    expect(result.current.theme.name).toEqual("Synnax Light");
  });
});
