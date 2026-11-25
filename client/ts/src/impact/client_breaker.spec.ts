// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Connector, HTTPClient } from "@synnaxlabs/freighter";
import { binary, errors, url } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { PlanChangedError } from "@/errors";
import { impact } from "@/impact";

interface Sent {
  target: string;
  body: unknown;
}

// fakeCore answers each request with the next status and body, as a Core over HTTP.
const fakeCore = (
  ...replies: Array<{ status: number; body: string }>
): { client: impact.Client; sent: Sent[] } => {
  const sent: Sent[] = [];
  const connector: Connector = {
    fetch: async (target, init) => {
      sent.push({
        target,
        body: JSON.parse(new TextDecoder().decode(init.body as Uint8Array)),
      });
      const reply = replies.shift();
      if (reply == null) throw new Error("fake Core has no reply left");
      const data = new TextEncoder().encode(reply.body);
      return {
        ok: reply.status >= 200 && reply.status < 300,
        status: reply.status,
        statusText: "",
        arrayBuffer: async () => data.buffer,
        body: null,
      };
    },
    dial: async () => {
      throw new Error("fake Core does not stream");
    },
  };
  const http = new HTTPClient({
    endpoint: new url.URL({ host: "localhost", port: 9999 }),
    codec: new binary.JSONCodec(),
    connector,
  });
  return { client: new impact.Client(http), sent };
};

const change: impact.Change = { type: "delete_channels", keys: [1048577] };

// What core/pkg/service/impact.Apply returns with no token, picks, or hidden pick,
// encoded by the Core's JSON codec: impact.Plan{}.
const EMPTY_APPLIED_PLAN =
  '{"entries":[],"kept":0,"hidden":0,"token":"","change":null}';

const CHANGED_PLAN = JSON.stringify({
  entries: [
    {
      record: { type: "task", key: "8c1b8c4e-6c1e-4a5e-9a39-4f1a3e0c2b11" },
      effect: "restrict",
      choices: ["cancel", "delete"],
      default: "cancel",
    },
  ],
  kept: 0,
  hidden: 0,
  token: "0123456789abcdef",
  change: { type: "delete_channels", keys: [1048577] },
});

describe("impact breaker", () => {
  it("should resolve an apply with no options after the Core applies it", async () => {
    const { client } = fakeCore({ status: 200, body: EMPTY_APPLIED_PLAN });
    await expect(client.apply(change)).resolves.toBeDefined();
  });

  it("should skip the token check when the token option is undefined", async () => {
    const { client, sent } = fakeCore({ status: 200, body: CHANGED_PLAN });
    const planned = undefined as impact.Plan | undefined;
    await expect(
      client.apply(change, { token: planned?.token }),
    ).resolves.toBeDefined();
    expect(sent[0].body).toMatchObject({ token: "" });
  });

  it("should keep the plan of a plan-changed error through encode and decode", () => {
    const decoded = errors.decode({ type: PlanChangedError.TYPE, data: CHANGED_PLAN });
    expect(decoded).toSatisfy(PlanChangedError.matches);
    const again = errors.decode(errors.encode(decoded)) as PlanChangedError;
    expect(again).toSatisfy(PlanChangedError.matches);
    expect(again.plan).toEqual(CHANGED_PLAN);
    expect(impact.planOf(again).token).toEqual("0123456789abcdef");
  });

  it("should send a pick key exactly as the caller names the record", async () => {
    const { client, sent } = fakeCore({ status: 200, body: CHANGED_PLAN });
    const record = "ni_device:01A2B3C4";
    await client.plan(change, { picks: { [record]: "delete" } });
    expect(sent[0].body).toMatchObject({ picks: { [record]: "delete" } });
  });
});
