// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { ICD } from "@/feature/icd";
import { client } from "@/feature/icd/testutil";
import { Session } from "@/session";
import { resolveFocusedTab, uniqueName } from "@/testutil";

describe("ICD commands", () => {
  it("should create an ICD and open it in a tab", async () => {
    const project = await client.projects.create({ name: uniqueName("project") });
    const { store, openCommandPalette } = await renderPalette({
      commands: ICD.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(project.key));
    await openCommandPalette();
    const item = await screen.findByText("Create ICD");
    await act(async () => {
      fireEvent.click(item);
    });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(icd.TYPE_RESOURCE_ID.type);
    const created = await client.icds.retrieve({ key: tab.resource.key });
    expect(created).toMatchObject({ name: "ICD", messages: [] });
  });
});
