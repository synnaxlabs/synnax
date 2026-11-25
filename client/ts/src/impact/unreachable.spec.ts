// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AddressInfo, createServer } from "node:net";

import { Unreachable } from "@synnaxlabs/freighter";
import { TimeSpan, url } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import type { channel } from "@/channel";
import { impact } from "@/impact";
import { Transport } from "@/transport";

const deleteOf = (...keys: channel.Key[]): impact.Change => impact.deleteChannels(keys);

describe("impact over an unreachable Core", () => {
  /** Returns a client whose transport points at a port that nothing listens on. */
  const createUnreachable = async (onRetry: (e: Error) => void) => {
    const server = createServer();
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const { port } = server.address() as AddressInfo;
    await new Promise<void>((resolve) => server.close(() => resolve()));
    const transport = new Transport({
      url: new url.URL({ host: "127.0.0.1", port }),
      retry: { baseInterval: TimeSpan.milliseconds(1), maxRetries: 2 },
      onRetry,
    });
    return new impact.Client({
      unary: transport.unary,
      unaryNoRetry: transport.unaryNoRetry,
    });
  };

  it("Should make one attempt for an apply and reject", async () => {
    const onRetry = vi.fn();
    const unreachable = await createUnreachable(onRetry);
    await expect(unreachable.apply(deleteOf(1))).rejects.toThrow(Unreachable);
    expect(onRetry).not.toHaveBeenCalled();
  });

  it("Should retry a plan before it rejects", async () => {
    const onRetry = vi.fn();
    const unreachable = await createUnreachable(onRetry);
    await expect(unreachable.plan(deleteOf(1))).rejects.toThrow(Unreachable);
    expect(onRetry).toHaveBeenCalledTimes(2);
  });
});
