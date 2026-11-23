// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { stubWidth, type Width } from "@/testutil";

const path = deep.path<{ config: { a: object; b: object } }>();

const Harness = () => {
  const [selected, setSelected] = useState<deep.Target<object>>(path.config.a);
  return (
    <Task.Views.Panes
      listTitle="Channels"
      list={<button onClick={() => setSelected(path.config.b)}>Select b</button>}
      detailsPath={selected}
    >
      <p>Details body</p>
    </Task.Views.Panes>
  );
};

const renderPanes = async (): Promise<HTMLElement> => {
  const { container } = await renderInTaskForm(<Harness />, {
    values: { config: { a: {}, b: {} } },
  });
  const panes = container.querySelector<HTMLElement>(".console-panes");
  if (panes == null) throw new Error("no panes");
  return panes;
};

const toggle = (panes: HTMLElement): void => {
  const button = panes.querySelector("[aria-expanded]");
  if (button == null) throw new Error("no list toggle");
  fireEvent.click(button);
};

const listOpen = (panes: HTMLElement): boolean =>
  panes.classList.contains("console--list-open");

describe("Task.Views.Panes", () => {
  let width: Width;

  beforeEach(() => {
    width = stubWidth("console-panes", 1000);
  });

  afterEach(() => width.restore());

  it("should hide and show the list in a wide layout", async () => {
    const panes = await renderPanes();
    expect(listOpen(panes)).toBe(true);
    toggle(panes);
    expect(listOpen(panes)).toBe(false);
    toggle(panes);
    expect(listOpen(panes)).toBe(true);
  });

  it("should open the list as a drawer in a narrow layout", async () => {
    const panes = await renderPanes();
    width.set(400);
    expect(panes.classList).toContain("console--narrow");
    expect(listOpen(panes)).toBe(false);
    toggle(panes);
    expect(listOpen(panes)).toBe(true);
  });

  it("should close the drawer when the selection changes", async () => {
    const panes = await renderPanes();
    width.set(400);
    toggle(panes);
    fireEvent.click(screen.getByText("Select b"));
    expect(listOpen(panes)).toBe(false);
  });

  it("should close the drawer on a click in the details", async () => {
    const panes = await renderPanes();
    width.set(400);
    toggle(panes);
    fireEvent.click(screen.getByText("Details body"));
    expect(listOpen(panes)).toBe(false);
  });

  it("should forget an open drawer when the layout widens", async () => {
    const panes = await renderPanes();
    width.set(400);
    toggle(panes);
    width.set(1000);
    toggle(panes);
    expect(listOpen(panes)).toBe(false);
    width.set(400);
    expect(listOpen(panes)).toBe(false);
  });
});
