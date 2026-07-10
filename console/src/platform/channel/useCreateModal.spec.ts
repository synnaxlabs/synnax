// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/platform/channel";
import { createTestIndexChannel, uniqueChannelName } from "@/platform/channel/testutil";
import {
  findButton,
  getSwitch,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { stubGeometry } from "@/testutil";

const client = createTestClient();

const openModal = async (): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Channel.useCreateModal, [], { client });
  await screen.findByPlaceholderText("Name");
  return handle;
};

const setName = (name: string): void => {
  fireEvent.change(screen.getByPlaceholderText("Name"), { target: { value: name } });
};

const clickCreate = async (): Promise<void> => {
  await act(async () => {
    fireEvent.click(findButton("Create"));
  });
};

stubGeometry();

describe("useCreateModal", () => {
  describe("index / virtual interaction", () => {
    it("should disable the index toggle once virtual is enabled", async () => {
      await openModal();
      expect(getSwitch("Is index").disabled).toBe(false);
      fireEvent.click(getSwitch("Virtual"));
      await waitFor(() => expect(getSwitch("Is index").disabled).toBe(true));
    });

    it("should disable the virtual toggle once is-index is enabled", async () => {
      await openModal();
      expect(getSwitch("Virtual").disabled).toBe(false);
      fireEvent.click(getSwitch("Is index"));
      await waitFor(() => expect(getSwitch("Virtual").disabled).toBe(true));
    });
  });

  describe("creation", () => {
    it("should create an index channel and close the modal", async () => {
      await openModal();
      const name = uniqueChannelName("idx");
      setName(name);
      fireEvent.click(getSwitch("Is index"));
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.isIndex).toBe(true);
        expect(created.dataType.equals(DataType.TIMESTAMP)).toBe(true);
      });
      await waitFor(() => expect(screen.queryByPlaceholderText("Name")).toBeNull());
    });

    it("should create a data channel paired with the selected index", async () => {
      const index = await createTestIndexChannel(client);
      await openModal();
      const name = uniqueChannelName("data");
      setName(name);
      fireEvent.click(screen.getByText("Select a channel"));
      fireEvent.change(await screen.findByPlaceholderText("Search channels...", {}), {
        target: { value: index.name },
      });
      fireEvent.click(await screen.findByText(index.name, {}));
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.index).toEqual(index.key);
        expect(created.isIndex).toBe(false);
      });
    });

    it("should create a virtual channel without an index", async () => {
      await openModal();
      const name = uniqueChannelName("virt");
      setName(name);
      fireEvent.click(getSwitch("Virtual"));
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.virtual).toBe(true);
        expect(created.index).toEqual(0);
      });
    });

    it("should keep the modal open and reset the form when create more is enabled", async () => {
      await openModal();
      const name = uniqueChannelName("more");
      setName(name);
      fireEvent.click(getSwitch("Is index"));
      fireEvent.click(getSwitch("Create More"));
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.isIndex).toBe(true);
      });
      await waitFor(() => {
        expect(screen.getByPlaceholderText<HTMLInputElement>("Name").value).toBe("");
      });
      expect(getSwitch("Is index").checked).toBe(false);
    });
  });

  describe("validation", () => {
    it("should keep the modal open and show an error when the name is empty", async () => {
      await openModal();
      fireEvent.click(getSwitch("Is index"));
      await clickCreate();
      await waitFor(() => expect(screen.getByText("Name is required")).toBeTruthy());
      expect(screen.getByPlaceholderText("Name")).toBeTruthy();
    });

    it("should reject a data channel that has no index and create nothing", async () => {
      await openModal();
      const name = uniqueChannelName("noidx");
      setName(name);
      await clickCreate();
      await waitFor(() =>
        expect(screen.getByText("Data channel must have an index")).toBeTruthy(),
      );
      expect(screen.getByPlaceholderText("Name")).toBeTruthy();
      const err = await client.channels.retrieve(name).then(
        () => null,
        (e: unknown) => e,
      );
      expect(NotFoundError.matches(err)).toBe(true);
    });
  });
});
