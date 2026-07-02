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
import { renderArc, TIMEOUT } from "@/platform/arc/testutil";
import { Session } from "@/session";

const Harness = (): ReactElement => {
  const create = Arc.useCreate();
  return <button onClick={() => create()}>open</button>;
};
Harness.displayName = "Harness";

const findArcLayout = (store: Session.Store, name: string) =>
  Object.values(store.getState()[Session.Layout.SLICE_NAME].layouts).find(
    (l) => l.type === Arc.LAYOUT_TYPE && l.name === name,
  );

describe("arc useCreate", () => {
  it("should open the create modal with a disabled Create button", async () => {
    await renderArc(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "open" }));
    await waitFor(
      () => expect(screen.getByPlaceholderText("Automation Name")).toBeTruthy(),
      TIMEOUT,
    );
    const create = screen.getByRole("button", { name: "Create" });
    expect(create.className).toContain("pluto--disabled");
  });

  it("should create the arc and place its layout after the modal is completed", async () => {
    const name = id.create();
    const { store } = await renderArc(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "open" }));

    const input = await screen.findByPlaceholderText("Automation Name");
    fireEvent.change(input, { target: { value: name } });

    const create = screen.getByRole("button", { name: "Create" });
    await waitFor(() => expect(create.className).not.toContain("pluto--disabled"));
    await act(async () => {
      fireEvent.click(create);
    });

    await waitFor(() => expect(findArcLayout(store, name)).toBeDefined(), TIMEOUT);
    const layout = findArcLayout(store, name);
    expect(layout?.type).toBe(Arc.LAYOUT_TYPE);
    expect(
      Session.Arc.selectState({ state: store.getState(), key: layout!.key }),
    ).toEqual(Session.Arc.ZERO_STATE);
  });
});
