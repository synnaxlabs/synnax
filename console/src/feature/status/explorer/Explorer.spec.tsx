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
import { Modals } from "@/platform/modals";
import { enableEditing } from "@/platform/view/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

describe("status explorer", () => {
  it("should list statuses matching a search", async () => {
    const s = await client.statuses.set(
      status.create({ name: uniqueName("status"), variant: "info", message: "m" }),
    );
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Status.Explorer.Explorer />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    await enableEditing();
    const search = await screen.findByPlaceholderText("Search statuses...");
    fireEvent.change(search, { target: { value: s.name } });
    expect(await screen.findByText(s.name)).toBeTruthy();
  });
});
