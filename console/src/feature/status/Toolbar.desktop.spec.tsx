// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Status } from "@/feature/status";
import { Notifications } from "@/platform/notifications";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

describe("status toolbar", () => {
  it("should show the favorites in their own tab", async () => {
    const { key, name } = await client.statuses.set(
      status.create({ name: uniqueName("status"), variant: "info", message: "" }),
    );
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Status.SLICE_NAME]: { version: 0, favorites: [key] },
      },
    });
    render(
      <Notifications.Provider rules={{}}>
        {Status.TOOLBAR.content}
      </Notifications.Provider>,
      {
        wrapper,
      },
    );
    fireEvent.click(screen.getByRole("tab", { name: "Favorites" }));
    expect(await screen.findByText(name)).toBeTruthy();
  });
});
