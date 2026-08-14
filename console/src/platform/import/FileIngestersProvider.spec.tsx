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

import { Import } from "@/platform/import";

const fileIngesters = { log: vi.fn(), table: vi.fn() };

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Import.FileIngestersProvider fileIngesters={fileIngesters}>
    {children}
  </Import.FileIngestersProvider>
);

describe("Import.FileIngestersProvider", () => {
  it("provides the file ingesters registry to descendants", () => {
    const { result } = renderHook(() => Import.useFileIngesters(), { wrapper });
    expect(result.current).toBe(fileIngesters);
  });

  it("throws when useFileIngesters is called outside of a provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => Import.useFileIngesters())).toThrow();
    spy.mockRestore();
  });
});
