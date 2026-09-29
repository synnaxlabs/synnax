// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("project/Splash", () => {
  it("should offer no log out and name no Core", async () => {
    const { store } = await renderWithConsole(<Project.Splash />);
    act(() => {
      store.dispatch(Session.Core.select(Session.Core.LOCAL_KEY));
    });
    const name = Session.Core.selectSelected(store.getState())?.name;
    if (name == null) throw new Error("no Core is selected");
    expect(screen.getByText("Projects")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Log out" })).toBeNull();
    expect(screen.queryByText(name)).toBeNull();
  });
});
