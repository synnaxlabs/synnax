// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/platform/panel";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  selectTestProject,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

describe("Panel.useOpenResource", () => {
  it("opens the resource's ontology ID as a tab", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const project = await selectTestProject(store, client);
    const created = await client.tables.create(project, { name: uniqueName("table") });
    const { result } = renderHook(() => Panel.useOpenResource(), { wrapper });
    const id = table.ontologyID(created.key);
    await act(async () => {
      result.current(ontology.resourceZ.parse({ id, name: created.name }));
    });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource).toEqual(id);
  });
});
