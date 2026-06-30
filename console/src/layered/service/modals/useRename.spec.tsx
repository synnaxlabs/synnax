// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, type RenderResult, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Modals } from "@/layered/service/modals";
import { renderWithModals } from "@/layered/service/modals/testutil";

let lastPrompt: Promise<string | null> | undefined;

const RenameHarness = ({ params }: { params: Modals.RenameParams }): ReactElement => {
  const rename = Modals.useRename();
  return (
    <button
      onClick={() => {
        lastPrompt = rename(params);
      }}
    >
      open
    </button>
  );
};
RenameHarness.displayName = "RenameHarness";

const openRename = (params: Modals.RenameParams = {}): RenderResult => {
  const result = renderWithModals(<RenameHarness params={params} />);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return result;
};

const getInput = (): HTMLInputElement => screen.getByRole("textbox");

const findCloseButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByRole("button")
    .find((b) => (b.textContent ?? "").trim() === "");
  if (btn == null) throw new Error("close button not found");
  return btn as HTMLButtonElement;
};

beforeEach(() => {
  lastPrompt = undefined;
});

describe("Rename", () => {
  it("should populate the input with the initial value", async () => {
    openRename({ initialValue: "original" });
    await waitFor(() => expect(getInput().value).toEqual("original"));
  });

  it("should resolve with the edited value when saved", async () => {
    openRename({ initialValue: "original" });
    const input = await screen.findByRole("textbox");
    fireEvent.change(input, { target: { value: "renamed" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await expect(lastPrompt).resolves.toEqual("renamed");
  });

  it("should disable saving when empty and empty values are not allowed", async () => {
    openRename({ allowEmpty: false });
    const save = await screen.findByRole("button", { name: "Save" });
    expect(save.className).toContain("pluto--disabled");
  });

  it("should resolve null when empty values are allowed and saved empty", async () => {
    openRename({ allowEmpty: true });
    fireEvent.click(await screen.findByRole("button", { name: "Save" }));
    await expect(lastPrompt).resolves.toBeNull();
  });

  it("should resolve null when dismissed via the close button", async () => {
    openRename({ initialValue: "original" });
    await screen.findByRole("textbox");
    fireEvent.click(findCloseButton());
    await expect(lastPrompt).resolves.toBeNull();
  });

  it("should render a custom title and label", async () => {
    openRename({ title: "Rename Range", label: "Range Name" });
    await waitFor(() => expect(screen.getByText("Rename Range")).toBeTruthy());
    expect(screen.getByText("Range Name")).toBeTruthy();
  });
});
