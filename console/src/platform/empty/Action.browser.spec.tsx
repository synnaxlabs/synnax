// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { server, userEvent } from "vitest/browser";

import { Empty } from "@/platform";

const renderAction = () => {
  const onClick = vi.fn();
  render(<Empty.Action message="No channels" action="Add channel" onClick={onClick} />);
  return { onClick, action: screen.getByRole("button", { name: "Add channel" }) };
};

describe("Empty.Action", () => {
  // WebKit tabs to a button only when the user turns on full keyboard access.
  it.runIf(server.browser === "chromium")("should take Tab focus", async () => {
    const { action } = renderAction();
    await userEvent.tab();
    expect(document.activeElement).toBe(action);
  });

  it("should run its action on Enter and Space", async () => {
    const { onClick, action } = renderAction();
    action.focus();
    await userEvent.keyboard("{Enter}");
    expect(onClick).toHaveBeenCalledTimes(1);
    await userEvent.keyboard(" ");
    expect(onClick).toHaveBeenCalledTimes(2);
  });
});
