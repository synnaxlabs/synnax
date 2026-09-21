// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Embedded } from "@/feature/embedded";
import { clearSupervisor, mockSupervisor, RUNNING } from "@/feature/embedded/testutil";

const PARAMS = {
  host: "127.0.0.1",
  port: 49152,
  username: "synnax",
  password: "launch-secret",
  name: "Synnax",
  secure: false,
};

describe("Embedded.useConnParams", () => {
  afterEach(clearSupervisor);

  it("should give no parameters before the Core runs", async () => {
    const { emitStatus } = mockSupervisor(() => ({ state: "starting" }));
    const { result } = renderHook(Embedded.useConnParams, {
      wrapper: Embedded.Provider,
    });
    expect(result.current).toBeUndefined();
    await act(async () => await emitStatus(RUNNING));
    await waitFor(() => expect(result.current).toEqual(PARAMS));
  });

  it("should read a Core that already runs when the window opens", async () => {
    mockSupervisor(() => RUNNING);
    const { result } = renderHook(Embedded.useConnParams, {
      wrapper: Embedded.Provider,
    });
    await waitFor(() => expect(result.current).toEqual(PARAMS));
  });

  it("should keep the parameters while the Core restarts", async () => {
    const { emitStatus } = mockSupervisor(() => RUNNING);
    const { result } = renderHook(Embedded.useConnParams, {
      wrapper: Embedded.Provider,
    });
    await waitFor(() => expect(result.current).toEqual(PARAMS));
    const before = result.current;
    await act(async () => await emitStatus({ state: "restarting" }));
    expect(result.current).toBe(before);
    await act(async () => await emitStatus(RUNNING));
    expect(result.current).toBe(before);
  });

  it("should prefer a status event over an older status answer", async () => {
    let answer: (status: Embedded.Status) => void = () => {};
    const pending = new Promise<Embedded.Status>((resolve) => (answer = resolve));
    const { emitStatus } = mockSupervisor(async () => await pending);
    const { result } = renderHook(Embedded.useConnParams, {
      wrapper: Embedded.Provider,
    });
    await act(async () => await emitStatus(RUNNING));
    await act(async () => answer({ state: "failed", message: "stale" }));
    expect(result.current).toEqual(PARAMS);
  });
});
