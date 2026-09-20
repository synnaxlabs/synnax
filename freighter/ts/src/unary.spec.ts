// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type breaker } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Unreachable } from "@/errors";
import { type UnaryClient, unaryWithBreaker } from "@/unary";

const NO_SLEEP: breaker.Config = { sleepFn: async () => {} };
const UNREACHABLE_MESSAGE = "server down";
const FATAL_MESSAGE = "malformed request";

const failingUnary = (error: Error): UnaryClient => ({
  send: vi.fn().mockRejectedValue(error),
  use: vi.fn(),
});

const send = async (client: UnaryClient): Promise<void> =>
  await client.send("check", undefined, z.void(), z.void());

describe("unaryWithBreaker", () => {
  let warn: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    warn = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => warn.mockRestore());

  it("should retry an unreachable target until the retries run out", async () => {
    const unary = failingUnary(new Unreachable({ message: UNREACHABLE_MESSAGE }));
    const client = unaryWithBreaker(unary, { ...NO_SLEEP, maxRetries: 2 });
    await expect(send(client)).rejects.toThrow(UNREACHABLE_MESSAGE);
    expect(unary.send).toHaveBeenCalledTimes(3);
    expect(warn).toHaveBeenCalledTimes(2);
  });

  it("should not announce a retry it will not make", async () => {
    const unary = failingUnary(new Unreachable({ message: UNREACHABLE_MESSAGE }));
    const client = unaryWithBreaker(unary, { ...NO_SLEEP, maxRetries: 0 });
    await expect(send(client)).rejects.toThrow(UNREACHABLE_MESSAGE);
    expect(unary.send).toHaveBeenCalledTimes(1);
    expect(warn).not.toHaveBeenCalled();
  });

  it("should rethrow an error that is not an unreachable target", async () => {
    const unary = failingUnary(new Error(FATAL_MESSAGE));
    const client = unaryWithBreaker(unary, { ...NO_SLEEP, maxRetries: 2 });
    await expect(send(client)).rejects.toThrow(FATAL_MESSAGE);
    expect(unary.send).toHaveBeenCalledTimes(1);
    expect(warn).not.toHaveBeenCalled();
  });
});
