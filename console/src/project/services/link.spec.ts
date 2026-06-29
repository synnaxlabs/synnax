// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { ProjectServices } from "@/project/services";
import { renderLinkHook } from "@/testUtils";

describe("ProjectServices.useLink", () => {
  it("should load the project layout and set it active", async () => {
    const key = "p1";
    const retrieve = vi.fn(async () => ({
      key,
      name: "Engine Project",
      layout: Layout.ZERO_SLICE_STATE,
    }));
    const client = { projects: { retrieve } } as unknown as Client;
    const { handler, store } = renderLinkHook(ProjectServices.useLink, {
      [Project.SLICE_NAME]: Project.reducer,
    });
    await handler({ client, key });
    expect(retrieve).toHaveBeenCalledWith(key);
    expect(Project.selectActiveKey(store.getState())).toBe(key);
  });
});
