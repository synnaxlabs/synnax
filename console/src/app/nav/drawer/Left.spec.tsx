// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createEmptyRole, createTestClientAs } from "@synnaxlabs/client/testutil";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { client, renderBar, withActiveProject } from "@/app/nav/bar/testutil";
import { Drawer } from "@/app/nav/drawer";
import { Session } from "@/session";
import { type ConsolePreloadedState, uniqueName } from "@/testutil";

// A user whose role holds no policies, so they may open no toolbar.
const createUnprivilegedClient = async () => {
  const { key: role } = await createEmptyRole(client);
  const username = uniqueName("user");
  await client.users.create({ username, password: "password123", role });
  return createTestClientAs(username, "password123");
};

const withSelected = (key: string): ConsolePreloadedState =>
  withActiveProject({
    [Session.Nav.SLICE_NAME]: {
      version: 0,
      windows: {
        [MAIN_WINDOW]: {
          ...Session.Nav.ZERO_WINDOW_STATE,
          left: { ...Session.Nav.ZERO_WINDOW_STATE.left, selected: key },
        },
      },
    },
  });

describe("app/nav/drawer/Left", () => {
  it("should render the selected toolbar for a subject who may open it", async () => {
    const { container } = await renderBar(<Drawer.Left />, withSelected("channel"));
    expect(await within(container).findByText("Channels")).toBeTruthy();
  });

  it("should collapse a toolbar the subject may not open", async () => {
    const { container } = await renderBar(
      <Drawer.Left />,
      withSelected("channel"),
      await createUnprivilegedClient(),
    );
    expect(within(container).queryByText("Channels")).toBeNull();
  });
});
