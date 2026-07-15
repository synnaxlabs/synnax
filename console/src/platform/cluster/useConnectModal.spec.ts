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

import { Cluster } from "@/platform/cluster";
import { createCluster, createClusterState } from "@/platform/cluster/testutil";
import {
  findButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { Session } from "@/session";
import { type ConsolePreloadedState } from "@/testutil";

const openConnect = async (
  args: [] | [Cluster.ConnectModalParams],
  preloadedState: ConsolePreloadedState = createClusterState([]),
): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Cluster.useConnectModal, args, {
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
    it("should prefill the form from the edited cluster", async () => {
      await openConnect(
        [{ clusterKey: "c1" }],
        createClusterState([
          createCluster("c1", { name: "Alpha", host: "alpha.host" }),
        ]),
      );
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
      expect(Session.Cluster.selectMany(store.getState())).toHaveLength(0);
    });

    it("should block submission and surface an error for a duplicate name", async () => {
      const { store } = await openConnect(
        [],
        createClusterState([createCluster("c1", { name: "Existing" })]),
      );
      fillForm("Existing", "localhost", "9090");
      fireEvent.click(findButton("Connect"));
      await waitFor(() =>
        expect(screen.getByText("Existing is already in use.")).toBeTruthy(),
      );
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy();
      expect(Session.Cluster.selectMany(store.getState())).toHaveLength(1);
    });
  });

  describe("submission", () => {
    it("should add the cluster and close the modal after connecting", async () => {
      const { store } = await openConnect([]);
      fillForm("SubmitTest", "localhost", "9090");
      fireEvent.click(findButton("Connect"));
      await waitFor(() => {
        const created = Session.Cluster.selectMany(store.getState()).find(
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

    it("should update the edited cluster, preserve credentials, and adopt the server's key on Save", async () => {
      const { store } = await openConnect(
        [{ clusterKey: "c1" }],
        createClusterState(
          [
            createCluster("c1", {
              name: "Alpha",
              username: "user_u",
              password: "pass_p",
            }),
          ],
          "c1",
        ),
      );
      fireEvent.change(nameInput(), { target: { value: "Alpha_Edited" } });
      fireEvent.click(findButton("Save"));
      await waitFor(() => {
        expect(Session.Cluster.selectState(store.getState(), "c1")).toBeUndefined();
        const edited = Session.Cluster.selectMany(store.getState()).find(
          (c) => c.name === "Alpha_Edited",
        );
        expect(edited).toBeDefined();
        expect(edited?.key).not.toEqual("c1");
        expect(edited?.username).toEqual("user_u");
        expect(edited?.password).toEqual("pass_p");
      });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Synnax Core")).toBeNull(),
      );
    });
  });
});
