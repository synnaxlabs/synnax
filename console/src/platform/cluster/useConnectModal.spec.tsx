// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Cluster } from "@/platform/cluster";
import { cluster, clusterState, renderClusterUI } from "@/platform/cluster/testutil";
import { Session } from "@/session";

const TIMEOUT = { timeout: 5000 };

const Harness = ({ params }: { params?: Cluster.ConnectModalParams }): ReactElement => {
  const open = Cluster.useConnectModal();
  return <button onClick={() => open(params)}>open</button>;
};
Harness.displayName = "Harness";

const openModal = async (
  params?: Cluster.ConnectModalParams,
  preloadedState = clusterState([]),
): Promise<void> => {
  await renderClusterUI(<Harness params={params} />, preloadedState);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  await waitFor(() => expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy());
};

describe("useConnectModal", () => {
  describe("create mode", () => {
    it("should render with an empty name and a Connect action", async () => {
      await openModal();
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("");
      expect(screen.getByRole("button", { name: "Connect" })).toBeTruthy();
    });
  });

  describe("edit mode", () => {
    it("should prefill the form and render a Save action", async () => {
      await openModal(
        { clusterKey: "c1" },
        clusterState([cluster("c1", { name: "Alpha", host: "alpha.host" })]),
      );
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      expect(nameInput.value).toEqual("Alpha");
      expect(screen.getByRole("button", { name: "Save" })).toBeTruthy();
    });
  });

  describe("validation", () => {
    it("should block submission and surface an error for a duplicate name", async () => {
      await openModal(undefined, clusterState([cluster("c1", { name: "Existing" })]));
      const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");
      fireEvent.change(nameInput, { target: { value: "Existing" } });
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
      await waitFor(() =>
        expect(screen.getByText("Existing is already in use.")).toBeTruthy(),
      );
      expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy();
    });
  });

  describe("submission", () => {
    it("should add the cluster and close the modal after connecting", async () => {
      const { store } = await renderClusterUI(<Harness />, clusterState([]));
      fireEvent.click(screen.getByRole("button", { name: "open" }));
      await waitFor(() =>
        expect(screen.getByPlaceholderText("Synnax Core")).toBeTruthy(),
      );
      fireEvent.change(screen.getByPlaceholderText("Synnax Core"), {
        target: { value: "SubmitTest" },
      });
      fireEvent.change(screen.getByPlaceholderText("localhost"), {
        target: { value: "localhost" },
      });
      fireEvent.change(screen.getByPlaceholderText("9090"), {
        target: { value: "9090" },
      });
      fireEvent.click(screen.getByRole("button", { name: "Connect" }));
      await waitFor(() => {
        const { clusters } = store.getState()[Session.Cluster.SLICE_NAME];
        expect(Object.values(clusters).some((c) => c.name === "SubmitTest")).toBe(true);
      }, TIMEOUT);
      await waitFor(
        () => expect(screen.queryByPlaceholderText("Synnax Core")).toBeNull(),
        TIMEOUT,
      );
    });
  });
});
