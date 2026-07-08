// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName, waitForPlacedLayout } from "@/testutil";

const client = createTestClient();

describe("log selectable", () => {
  it("should create a log under the layout key when clicked", async () => {
    const project = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const layoutKey = uuid.create();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: project.key },
      },
    });
    render(
      <Log.Selectable
        layoutKey={layoutKey}
        rename={async () => null}
        onPlace={() => ({ windowKey: "main", key: layoutKey })}
        handleError={() => {}}
      />,
      { wrapper },
    );
    fireEvent.click(await screen.findByText("Log"));
    const placedKey = await waitForPlacedLayout(store, Log.LAYOUT_TYPE);
    const created = await client.logs.retrieve({ key: placedKey });
    expect(created.name).toBe("Log");
  });
});
