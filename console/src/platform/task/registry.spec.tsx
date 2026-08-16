// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";

const registry: Task.Registry = {
  getIcon: () => <Icon.Task />,
  parseType: (type: string) => type.replace("ni_", ""),
};

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Task.RegistryProvider registry={registry}>{children}</Task.RegistryProvider>
);

describe("registry", () => {
  it("should expose the provided registry to consumers", () => {
    const { result } = renderHook(() => Task.useRegistry(), { wrapper });
    expect(result.current).toBe(registry);
  });

  it("should throw when used outside of a provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => Task.useRegistry())).toThrow();
    spy.mockRestore();
  });
});
