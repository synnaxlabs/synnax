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

import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Channel } from "@/platform/channel";
import { createTestIndexChannel, uniqueChannelName } from "@/platform/channel/testutil";
import {
  findButton,
  getSwitch,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";

const client = createTestClient();

const openModal = async (
  analytics?: Analytics.Sink,
): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Channel.useCreateModal, [], {
    client,
    analytics,
  });
  await screen.findByPlaceholderText("Channel name");
  return handle;
};

const setName = (name: string): void => {
  fireEvent.change(screen.getByPlaceholderText("Channel name"), {
    target: { value: name },
  });
};

const getKind = (name: string): HTMLElement => screen.getByRole("button", { name });

const selectKind = (name: string): void => {
  fireEvent.click(getKind(name));
};

const clickCreate = async (): Promise<void> => {
  await act(async () => {
    fireEvent.click(findButton("Create"));
  });
};

describe("useCreateModal", () => {
  describe("kind", () => {
    it("should start as a data channel that asks for an index", async () => {
      await openModal();
      expect(getKind("Data").getAttribute("aria-pressed")).toBe("true");
      expect(screen.getByText("Select channel")).toBeTruthy();
    });

    it("should hide the index field for an index channel", async () => {
      await openModal();
      selectKind("Index");
      await waitFor(() => expect(screen.queryByText("Select channel")).toBeNull());
    });

    it("should hide the index field for a virtual channel", async () => {
      await openModal();
      selectKind("Virtual");
      await waitFor(() => expect(screen.queryByText("Select channel")).toBeNull());
    });

    it("should restore the default data type when leaving the index kind", async () => {
      await openModal();
      const name = uniqueChannelName("virt");
      setName(name);
      selectKind("Index");
      selectKind("Virtual");
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.isIndex).toBe(false);
        expect(created.dataType.equals(DataType.FLOAT32)).toBe(true);
      });
    });
  });

  describe("creation", () => {
    it("should create an index channel and close the modal", async () => {
      await openModal();
      const name = uniqueChannelName("idx");
      setName(name);
      selectKind("Index");
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.isIndex).toBe(true);
        expect(created.dataType.equals(DataType.TIMESTAMP)).toBe(true);
      });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Channel name")).toBeNull(),
      );
    });

    it("should create a data channel paired with the selected index", async () => {
      const index = await createTestIndexChannel(client);
      await openModal();
      const name = uniqueChannelName("data");
      setName(name);
      fireEvent.click(screen.getByText("Select channel"));
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
      selectKind("Virtual");
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.virtual).toBe(true);
        expect(created.index).toEqual(0);
      });
    });

    it("should report the channel it created", async () => {
      const analytics = createTestSink();
      await openModal(analytics);
      setName(uniqueChannelName("virt"));
      selectKind("Virtual");
      await clickCreate();
      await waitFor(() =>
        expect(analytics.capture).toHaveBeenCalledWith("channel_created", {
          calculated: false,
        }),
      );
    });

    it("should keep the modal open and reset the form when create more is enabled", async () => {
      await openModal();
      const name = uniqueChannelName("more");
      setName(name);
      selectKind("Index");
      fireEvent.click(getSwitch("Create more"));
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.isIndex).toBe(true);
      });
      await waitFor(() => {
        expect(
          screen.getByPlaceholderText<HTMLInputElement>("Channel name").value,
        ).toBe("");
      });
      expect(getKind("Data").getAttribute("aria-pressed")).toBe("true");
    });
  });

  describe("validation", () => {
    it("should keep the modal open and show an error when the name is empty", async () => {
      await openModal();
      selectKind("Index");
      await clickCreate();
      await waitFor(() => expect(screen.getByText("Name is required")).toBeTruthy());
      expect(screen.getByPlaceholderText("Channel name")).toBeTruthy();
    });

    it("should reject a data channel that has no index and create nothing", async () => {
      await openModal();
      const name = uniqueChannelName("noidx");
      setName(name);
      await clickCreate();
      await waitFor(() =>
        expect(screen.getByText("Data channel must have an index")).toBeTruthy(),
      );
      expect(screen.getByPlaceholderText("Channel name")).toBeTruthy();
      const err = await client.channels.retrieve(name).then(
        () => null,
        (e: unknown) => e,
      );
      expect(NotFoundError.matches(err)).toBe(true);
    });
  });
});
