// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Core } from "@/platform/core";
import {
  findButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { type ConsolePreloadedState } from "@/testutil";

const openConnect = async (
  args: [] | [Core.ConnectModalParams],
  preloadedState: ConsolePreloadedState = createCoreState([]),
): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Core.useConnectModal, args, {
    preloadedState,
  });
  await waitFor(() => expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy());
  return handle;
};

const nameInput = (): HTMLInputElement =>
  screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");

const fillForm = (name: string, host: string, port: string): void => {
  fireEvent.change(nameInput(), { target: { value: name } });
  fireEvent.change(screen.getByPlaceholderText("localhost"), {
    target: { value: host },
  });
  fireEvent.change(screen.getByPlaceholderText("9090"), { target: { value: port } });
};

describe("useConnectModal", () => {
  describe("edit mode", () => {
    it("should prefill the form from the edited Core", async () => {
      const alpha = createCore("Alpha", { host: "alpha.host" });
      await openConnect([{ coreKey: alpha.key }], createCoreState([alpha]));
      expect(nameInput().value).toEqual("Alpha");
      expect(screen.getByPlaceholderText<HTMLInputElement>("localhost").value).toEqual(
        "alpha.host",
      );
      expect(findButton("Save")).toBeTruthy();
    });
  });

  describe("validation", () => {
    it("should keep the modal open and dispatch nothing when required fields are empty", async () => {
      const { store } = await openConnect([]);
      fireEvent.click(findButton("Connect"));
      await waitFor(() => expect(screen.getByText("Host is required")).toBeTruthy());
      expect(screen.getByText("Name is required")).toBeTruthy();
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy();
      expect(Session.Core.selectMany(store.getState())).toHaveLength(0);
    });

    it("should block submission and surface an error for a duplicate name", async () => {
      const { store } = await openConnect(
        [],
        createCoreState([createCore("c1", { name: "Existing" })]),
      );
      fillForm("Existing", "localhost", "9090");
      fireEvent.click(findButton("Connect"));
      await waitFor(() =>
        expect(screen.getByText("Existing is already in use.")).toBeTruthy(),
      );
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy();
      expect(Session.Core.selectMany(store.getState())).toHaveLength(1);
    });
  });

  describe("submission", () => {
    it("should add the Core and close the modal after connecting", async () => {
      const { store } = await openConnect([]);
      fillForm("SubmitTest", "localhost", "9090");
      fireEvent.click(findButton("Connect"));
      await waitFor(() => {
        const created = Session.Core.selectMany(store.getState()).find(
          (c) => c.name === "SubmitTest",
        );
        expect(created).toBeDefined();
        expect(created?.host).toEqual("localhost");
        expect(created?.port).toEqual("9090");
      });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Synnax Core")).toBeNull(),
      );
    });

    it("should rename the edited Core in place and preserve its credentials", async () => {
      const alpha = createCore("Alpha", { username: "user_u", password: "pass_p" });
      const { store } = await openConnect(
        [{ coreKey: alpha.key }],
        createCoreState([alpha], alpha.key),
      );
      fireEvent.change(nameInput(), { target: { value: "Alpha_Edited" } });
      fireEvent.click(findButton("Save"));
      await waitFor(() => {
        const edited = Session.Core.selectState(store.getState(), alpha.key);
        expect(edited?.name).toEqual("Alpha_Edited");
        // The address did not change, so neither did the key it implies.
        expect(edited?.key).toEqual(alpha.key);
        expect(edited?.username).toEqual("user_u");
        expect(edited?.password).toEqual("pass_p");
      });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Synnax Core")).toBeNull(),
      );
    });
  });
});
