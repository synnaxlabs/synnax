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

import { Project } from "@/feature/project";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("Project.useLink", () => {
  it("should load the project layout and set it active", async () => {
    const project = await client.projects.create({
      name: "Engine Project",
      layout: {},
    });
    const { handler, store } = await renderLinkHook(Project.useLink);
    await handler({ client, key: project.key });
    expect(Session.Project.selectSelected(store.getState())).toBe(project.key);
  });
});
