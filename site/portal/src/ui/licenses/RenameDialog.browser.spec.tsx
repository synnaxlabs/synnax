// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { activationOf, HASH_A } from "@/server/license/testutil";
import { Licenses } from "@/ui/licenses";

describe("RenameDialog", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("should fill the name with the machine's current name", () => {
    render(
      <Licenses.RenameDialog
        activation={activationOf([HASH_A], { name: "Test stand" })}
        visible
      />,
    );
    expect(screen.getByDisplayValue("Test stand")).toBeTruthy();
  });

  it("should reject an empty name without posting it", async () => {
    const fetch = vi.spyOn(globalThis, "fetch");
    render(
      <Licenses.RenameDialog
        activation={activationOf([HASH_A], { name: "Test stand" })}
        visible
      />,
    );
    fireEvent.change(screen.getByDisplayValue("Test stand"), {
      target: { value: " " },
    });
    fireEvent.click(screen.getByText("Rename"));
    await waitFor(() => expect(screen.getByText("Name the machine")).toBeTruthy());
    expect(fetch).not.toHaveBeenCalled();
  });
});
