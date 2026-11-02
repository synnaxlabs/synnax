// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Core } from "@/platform/core";
import { serveAuth } from "@/platform/core/testutil";
import { disableActEnvironment } from "@/testutil";

// the local server answers outside any act
disableActEnvironment();

describe("useAcceptedMethods", () => {
  it("should accept passwords before a Core answers", () => {
    const { result } = renderHook(() => Core.useAcceptedMethods());
    expect(result.current).toEqual({ password: true, oidc: false });
  });

  it("should report the methods of the Core", async () => {
    const core = await serveAuth(["oidc", "api_key"]);
    const { result } = renderHook(() => Core.useAcceptedMethods(core));
    await waitFor(() =>
      expect(result.current).toEqual({ password: false, oidc: true }),
    );
  });

  it("should keep passwords when the methods cannot be read", async () => {
    const core = await serveAuth(["oidc"]);
    const { result, rerender } = renderHook(
      ({ port }) => Core.useAcceptedMethods({ host: core.host, port }),
      { initialProps: { port: core.port } },
    );
    await waitFor(() => expect(result.current.password).toBe(false));
    rerender({ port: 1 });
    await waitFor(() => expect(result.current.password).toBe(true));
  });
});
