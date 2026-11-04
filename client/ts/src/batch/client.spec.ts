// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { describe, expect, it } from "vitest";

import { batch } from "@/batch";
import { channel } from "@/channel";
import { group } from "@/group";

const G1 = "11111111-1111-4111-8111-111111111111";

describe("batch client", () => {
  it("should send every request to the batch route in order", async () => {
    const sent: Array<{ target: string; req: unknown }> = [];
    const unary: UnaryClient = {
      use: () => {},
      send: async (target, req, reqZ, resZ) => {
        sent.push({ target, req: reqZ.parse(req) });
        return resZ.parse({ responses: [{ group: { key: G1 } }, {}] });
      },
    };
    const requests = [
      group.createRequest({ key: G1, name: "Pumps", scope: "channel" }),
      channel.setGroupRequest([1, 2], G1),
    ];
    const responses = await new batch.Client(unary).send(requests);
    expect(sent).toEqual([
      {
        target: "/batch",
        req: {
          requests: [
            {
              endpoint: "/group/create",
              payload: { key: G1, name: "Pumps", scope: "channel" },
            },
            { endpoint: "/channel/set-group", payload: { keys: [1, 2], group: G1 } },
          ],
        },
      },
    ]);
    expect(responses).toEqual([{ group: { key: G1 } }, {}]);
  });
});
