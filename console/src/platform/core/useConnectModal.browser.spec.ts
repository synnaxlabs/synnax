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
import { connectNameInput, fillConnectForm } from "@/platform/core/testutil";
import {
  findButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import type { ConsolePreloadedState } from "@/testutil";

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

describe("useConnectModal", () => {
  describe("edit mode", () => {
    it("should prefill the form from the edited Core", async () => {
      const alpha = createCore("Alpha", { host: "alpha.host" });
      await openConnect([{ coreKey: alpha.key }], createCoreState([alpha]));
      expect(connectNameInput().value).toEqual("Alpha");
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
      expect(
        Object.values(store.getState()[Session.Core.SLICE_NAME].cores),
      ).toHaveLength(0);
    });

    // A name is a label, not an identity, so two Cores may share one.
    it("should add a Core whose name another Core already uses", async () => {
      const { store } = await openConnect(
        [],
        createCoreState([createCore("c1", { name: "Existing" })]),
      );
      fillConnectForm("Existing", "localhost", "9090");
      fireEvent.click(findButton("Connect"));
      await waitFor(() =>
        expect(
          Object.values(store.getState()[Session.Core.SLICE_NAME].cores),
        ).toHaveLength(2),
      );
    });
  });

  describe("submission", () => {
    it("should add the Core and close the modal after connecting", async () => {
      const { store } = await openConnect([]);
      fillConnectForm("SubmitTest", "localhost", "9091");
      fireEvent.click(findButton("Connect"));
      await waitFor(() => {
        const created = Object.values(
          store.getState()[Session.Core.SLICE_NAME].cores,
        ).find((c) => c.name === "SubmitTest");
        expect(created).toBeDefined();
        expect(created?.host).toEqual("localhost");
        expect(created?.port).toEqual(9091);
      });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Synnax Core")).toBeNull(),
      );
    });

    it("should rename the edited Core in place and preserve its login", async () => {
      const alpha = createCore("Alpha", {
        username: "user_u",
        password: "pass_p",
        lastLogin: 1234,
      });
      const { store } = await openConnect(
        [{ coreKey: alpha.key }],
        createCoreState([alpha], alpha.key),
      );
      fireEvent.change(connectNameInput(), { target: { value: "Alpha_Edited" } });
      fireEvent.click(findButton("Save"));
      await waitFor(() => {
        const edited = Session.Core.selectState(store.getState(), alpha.key);
        expect(edited?.name).toEqual("Alpha_Edited");
        // The address did not change, so neither did the key it implies.
        expect(edited?.key).toEqual(alpha.key);
        expect(edited?.username).toEqual("user_u");
        expect(edited?.password).toEqual("pass_p");
        // a new lastLogin would give the Core a new client
        expect(edited?.lastLogin).toEqual(1234);
      });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Synnax Core")).toBeNull(),
      );
    });
  });
});
