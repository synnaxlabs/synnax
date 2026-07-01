// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Log } from "@/service/log";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil/testutil";

const client = createTestClient();

describe("Log.useLink", () => {
  it("should place a log layout for the retrieved log", async () => {
    const project = await client.projects.create({ name: id.create(), layout: {} });
    const log = await client.logs.create(project.key, { name: "Event Log" });
    const { handler, store } = renderLinkHook(Log.useLink);
    await handler({ client, key: log.key });
    expect(Session.Layout.select(store.getState(), log.key)?.name).toBe("Event Log");
  });
});
