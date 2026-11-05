// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { video } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Video } from "@/feature/video";
import { findCommand } from "@/platform/command/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  renderHookWithConsole,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("Video Commands", () => {
  it("should create a video view in the active project and open it", async () => {
    const project = await client.projects.create({ name: uniqueName("project") });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Video.COMMANDS,
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: project.key },
      },
    });
    await openCommandPalette();
    await selectCommand("Create video view");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(video.TYPE_ONTOLOGY_ID.type);
    const created = await client.videos.retrieve({ key: tab.resource.key });
    expect(created.name).toBe("Video");
  });

  it("should offer the create command to an engineer", async () => {
    const gate = findCommand(Video.COMMANDS, "Create video view").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold the create command from a viewer", async () => {
    const gate = findCommand(Video.COMMANDS, "Create video view").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(video.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
