// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("Arc.useLink", () => {
  it("should place an arc layout for the retrieved arc", async () => {
    const arc = await client.arcs.create({
      name: "Control Sequence",
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const { handler, store } = await renderLinkHook(Arc.useLink);
    await handler({ client, key: arc.key });
    expect(Session.Layout.select(store.getState(), arc.key)?.name).toBe(
      "Control Sequence",
    );
  });
});
