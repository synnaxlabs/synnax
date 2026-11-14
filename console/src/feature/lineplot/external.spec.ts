// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("LinePlot.TABS restore", () => {
  it("should restore a deleted line plot into the given project", async () => {
    const { restore } = LinePlot.TABS[lineplot.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const from = await client.projects.create({ name: uniqueName("from") });
    const to = await client.projects.create({ name: uniqueName("to") });
    const created = await client.lineplots.create({
      project: from.key,
      name: uniqueName("line plot"),
    });
    await client.lineplots.retrieve(created.key);
    await client.lineplots.delete(created.key);
    await waitFor(() =>
      expect(
        query.Deleted.matches(client.lineplots.watch(created.key).getCached()),
      ).toBe(true),
    );
    await restore({
      client,
      project: to.key,
      resource: lineplot.resourceID(created.key),
    });
    const restored = await createTestClient().lineplots.retrieve(created.key);
    expect(restored.project).toEqual(to.key);
  });
});
