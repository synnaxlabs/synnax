// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { client, renderArc } from "@/platform/arc/testutil";

const Harness = (): ReactElement => {
  const create = Arc.useCreate();
  return <button onClick={() => create()}>open</button>;
};
Harness.displayName = "Harness";

describe("arc useCreate", () => {
  it("should open the create modal with a disabled Create button", async () => {
    await renderArc(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "open" }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Automation Name")).toBeTruthy(),
    );
    const create = screen.getByRole("button", { name: "Create" });
    expect(create.className).toContain("pluto--disabled");
  });

  it("should create the arc on the server after the modal completes", async () => {
    const name = id.create();
    await renderArc(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "open" }));

    const input = await screen.findByPlaceholderText("Automation Name");
    fireEvent.change(input, { target: { value: name } });

    const create = screen.getByRole("button", { name: "Create" });
    await waitFor(() => expect(create.className).not.toContain("pluto--disabled"));
    await act(async () => {
      fireEvent.click(create);
    });

    const retrieved = await waitFor(async () => await client.arcs.retrieve({ name }));
    expect(retrieved.name).toBe(name);
  });
});
