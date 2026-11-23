// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Portal } from "@/portal";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const Counter = (): ReactElement => {
  const [count, setCount] = useState(0);
  return <button onClick={() => setCount((c) => c + 1)}>{`count:${count}`}</button>;
};

const Modal = ({ children }: { children?: ReactElement }): ReactElement => (
  <Dialog.Frame variant="modal">
    <Dialog.Trigger>Open</Dialog.Trigger>
    <Dialog.Dialog>
      <p>Modal</p>
      {children}
    </Dialog.Dialog>
  </Dialog.Frame>
);

const renderAbove = async (modal: ReactElement): Promise<HTMLElement> => {
  const host = document.createElement("div");
  document.body.appendChild(host);
  await renderThemed(
    <Triggers.Provider>
      <Portal.Above element={host}>
        <Counter />
      </Portal.Above>
      {modal}
    </Triggers.Provider>,
  );
  return host;
};

const counter = (): Element => page.getByRole("button", { name: /count/ }).element();

describe("Portal.Above", () => {
  it("should render into its element while no modal is open", async () => {
    const host = await renderAbove(<Modal />);
    expect(host.contains(counter())).toBe(true);
  });

  it("should move into an open modal and stay usable there", async () => {
    await renderAbove(<Modal />);
    await userEvent.click(page.getByRole("button", { name: "Open" }));
    const dialog = page.getByRole("dialog").element();
    expect(dialog.contains(counter())).toBe(true);
    await userEvent.click(counter());
    await expect.element(page.getByText("count:1")).toBeVisible();
    expect(page.getByText("Modal").query()).not.toBeNull();
  });

  it("should go back to its element with its state when the modal closes", async () => {
    const host = await renderAbove(<Modal />);
    await userEvent.click(page.getByRole("button", { name: "Open" }));
    await userEvent.click(counter());
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByText("Modal")).not.toBeInTheDocument();
    expect(host.contains(page.getByText("count:1").element())).toBe(true);
  });

  it("should move into a modal opened from another modal", async () => {
    await renderAbove(
      <Modal>
        <Dialog.Frame variant="modal">
          <Dialog.Trigger>Inner</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Inner modal</p>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Modal>,
    );
    await userEvent.click(page.getByRole("button", { name: "Open" }));
    await userEvent.click(page.getByRole("button", { name: "Inner" }));
    const inner = page.getByText("Inner modal").element().closest("dialog");
    expect(inner?.contains(counter())).toBe(true);
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByText("Inner modal")).not.toBeInTheDocument();
    const outer = page.getByText("Modal").element().closest("dialog");
    expect(outer?.contains(counter())).toBe(true);
  });
});
