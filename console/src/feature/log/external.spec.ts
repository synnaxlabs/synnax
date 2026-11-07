// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeStamp } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Log.TABS restore", () => {
  it("should restore a deleted log of a range into a project", async () => {
    const { restore } = Log.TABS[log.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const rng = await client.ranges.create({
      name: uniqueName("range"),
      timeRange: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2)),
    });
    const to = await client.projects.create({ name: uniqueName("to") });
    const created = await client.logs.create({
      range: rng.key,
      name: uniqueName("log"),
    });
    await client.logs.retrieve(created.key);
    await client.logs.delete(created.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.logs.getCached(created.key))).toBe(true),
    );
    await restore({ client, project: to.key, resource: log.resourceID(created.key) });
    const restored = await createTestClient().logs.retrieve(created.key);
    expect(restored.project).toEqual(to.key);
    expect(restored.range).toBeUndefined();
  });
});
