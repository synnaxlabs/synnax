// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Synnax } from "@/synnax";

const LIVE_PARAMS: connection.CheckParams = {
  host: TEST_CLIENT_PARAMS.host,
  port: TEST_CLIENT_PARAMS.port,
};

describe("useCheckConnection", () => {
  it("should return null when params are absent", () => {
    const { result } = renderHook(() => Synnax.useCheckConnection(null));
    expect(result.current).toBeNull();
  });

  it("should resolve a success status against a live cluster", async () => {
    const { result } = renderHook(() => Synnax.useCheckConnection(LIVE_PARAMS));
    expect(result.current).toBeNull();
    await waitFor(() => expect(result.current?.variant).toEqual("success"));
  });

  it("should resolve an error status against a dead port", async () => {
    const { result } = renderHook(() =>
      Synnax.useCheckConnection({
        host: TEST_CLIENT_PARAMS.host,
        port: 9999,
        retry: { maxRetries: 0 },
      }),
    );
    await waitFor(() => expect(result.current?.variant).toEqual("error"));
  });

  it("should keep checking on the interval while mounted", async () => {
    const { result } = renderHook(() =>
      Synnax.useCheckConnection(LIVE_PARAMS, TimeSpan.milliseconds(10)),
    );
    await waitFor(() => expect(result.current?.variant).toEqual("success"));
    const first = result.current;
    // A later poll replaces the status object, proving the loop keeps running.
    await waitFor(() => {
      expect(result.current?.variant).toEqual("success");
      expect(result.current).not.toBe(first);
    });
  });

  it("should not re-run the check for structurally equal params", async () => {
    const { result, rerender } = renderHook(
      ({ params }: { params: connection.CheckParams }) =>
        Synnax.useCheckConnection(params),
      { initialProps: { params: LIVE_PARAMS } },
    );
    await waitFor(() => expect(result.current?.variant).toEqual("success"));
    rerender({ params: { ...LIVE_PARAMS } });
    expect(result.current?.variant).toEqual("success");
  });

  it("should re-run the check when params change", async () => {
    const { result, rerender } = renderHook(
      ({ params }: { params: connection.CheckParams }) =>
        Synnax.useCheckConnection(params),
      { initialProps: { params: LIVE_PARAMS } },
    );
    await waitFor(() => expect(result.current?.variant).toEqual("success"));
    rerender({
      params: { host: TEST_CLIENT_PARAMS.host, port: 9999, retry: { maxRetries: 0 } },
    });
    await waitFor(() => expect(result.current?.variant).toEqual("error"));
  });
});
