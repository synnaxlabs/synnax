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
import { describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";

const extractors = { log: vi.fn(), table: vi.fn() };

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Export.ExtractorsProvider extractors={extractors}>
    {children}
  </Export.ExtractorsProvider>
);

describe("Export.ExtractorsProvider", () => {
  it("provides the extractors registry to descendants", () => {
    const { result } = renderHook(() => Export.useExtractors(), { wrapper });
    expect(result.current).toBe(extractors);
  });

  it("throws when useExtractors is called outside of a provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => Export.useExtractors())).toThrow();
    spy.mockRestore();
  });
});
