// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, group, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import type { Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Channel } from "@/platform/channel";
import { createTestIndexChannel } from "@/platform/channel/testutil";
import {
  findButton,
  getSwitch,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { CaptureStatuses, pickOption } from "@/testutil";

const client = createTestClient();

interface OpenModalOptions {
  analytics?: Analytics.Sink;
  params?: Channel.CreateParams;
  /** Receives the status notifications on each change. */
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

const openModal = async ({
  analytics,
  params,
  onStatuses,
}: OpenModalOptions = {}): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(
    Channel.useCreateModal,
    params == null ? [] : [params],
    {
      client,
      analytics,
      extra:
        onStatuses == null ? undefined : createElement(CaptureStatuses, { onStatuses }),
    },
  );
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
      const name = uniqueName("virt");
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
      const name = uniqueName("idx");
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
      const name = uniqueName("data");
      setName(name);
      fireEvent.click(screen.getByText("Select channel"));
      await pickOption(index.name, "Search channels...");
      await clickCreate();
      await waitFor(async () => {
        const created = await client.channels.retrieve(name);
        expect(created.index).toEqual(index.key);
        expect(created.isIndex).toBe(false);
      });
    });

    it("should create a virtual channel without an index", async () => {
      await openModal();
      const name = uniqueName("virt");
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
      await openModal({ analytics });
      setName(uniqueName("virt"));
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
      const name = uniqueName("more");
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

  describe("keys", () => {
    const press = (target: Element, key: string): void => {
      fireEvent.keyDown(target, { key, code: key });
      fireEvent.keyUp(target, { key, code: key });
    };

    it("should keep the kind when the arrow keys and Enter pick an index", async () => {
      const index = await createTestIndexChannel(client);
      await openModal();
      fireEvent.click(screen.getByText("Select channel"));
      const search = await screen.findByPlaceholderText("Search channels...");
      fireEvent.change(search, { target: { value: index.name } });
      await screen.findByText(index.name);
      press(search, "ArrowDown");
      press(search, "Enter");
      await waitFor(() => expect(screen.queryByText("Select channel")).toBeNull());
      expect(getKind("Data").getAttribute("aria-pressed")).toBe("true");
      expect(screen.getByText(index.name)).toBeTruthy();
    });

    it("should pick a kind once with Enter on its focused button", async () => {
      await openModal();
      selectKind("Data");
      const index = getKind("Index");
      index.focus();
      await userEvent.keyboard("{Enter}");
      await waitFor(() => expect(index.getAttribute("aria-pressed")).toBe("true"));
      expect(getKind("Data").getAttribute("aria-pressed")).toBe("false");
      expect(screen.queryByText("Select channel")).toBeNull();
    });

    it("should keep the index when Enter lands on the kind that is already picked", async () => {
      const index = await createTestIndexChannel(client);
      await openModal();
      fireEvent.click(screen.getByText("Select channel"));
      await pickOption(index.name, "Search channels...");
      const data = getKind("Data");
      data.focus();
      await userEvent.keyboard("{ArrowDown}{Enter}");
      await waitFor(() => expect(data.getAttribute("aria-pressed")).toBe("true"));
      expect(screen.getByText(index.name)).toBeTruthy();
    });
  });

  describe("folders", () => {
    const createFolder = async (): Promise<group.Group> =>
      await client.groups.create({ name: uniqueName("folder"), scope: "channel" });

    const createVirtual = (name: string): void => {
      setName(name);
      selectKind("Virtual");
    };

    it("should create the channel in the folder that opens the modal", async () => {
      const folder = await createFolder();
      await openModal({ params: { group: folder.key } });
      expect(await screen.findByText(folder.name)).toBeTruthy();
      const name = uniqueName("virt");
      createVirtual(name);
      await clickCreate();
      await waitFor(async () =>
        expect((await client.channels.retrieve(name)).group).toEqual(folder.key),
      );
    });

    it("should create the channel in the folder picked in the modal", async () => {
      const folder = await createFolder();
      await openModal();
      fireEvent.click(screen.getByText("Select folder"));
      await pickOption(folder.name, "Search folders...");
      const name = uniqueName("virt");
      createVirtual(name);
      await clickCreate();
      await waitFor(async () =>
        expect((await client.channels.retrieve(name)).group).toEqual(folder.key),
      );
    });

    it("should create a channel at the top when no folder is picked", async () => {
      await openModal();
      const name = uniqueName("virt");
      createVirtual(name);
      await clickCreate();
      await waitFor(async () =>
        expect((await client.channels.retrieve(name)).group).toBeUndefined(),
      );
    });

    it("should keep the folder for the next channel when create more is enabled", async () => {
      const folder = await createFolder();
      await openModal({ params: { group: folder.key } });
      fireEvent.click(getSwitch("Create more"));
      const first = uniqueName("virt");
      createVirtual(first);
      await clickCreate();
      await waitFor(async () =>
        expect((await client.channels.retrieve(first)).group).toEqual(folder.key),
      );
      await waitFor(() =>
        expect(
          screen.getByPlaceholderText<HTMLInputElement>("Channel name").value,
        ).toBe(""),
      );
      const second = uniqueName("virt");
      createVirtual(second);
      await clickCreate();
      await waitFor(async () =>
        expect((await client.channels.retrieve(second)).group).toEqual(folder.key),
      );
    });

    it("should refuse a name that a channel in the folder holds and create nothing", async () => {
      const folder = await createFolder();
      const taken = await client.channels.create({
        name: uniqueName("taken"),
        dataType: DataType.FLOAT32,
        virtual: true,
        group: folder.key,
      });
      let statuses: Status.NotificationSpec[] = [];
      await openModal({
        params: { group: folder.key },
        onStatuses: (next) => (statuses = next),
      });
      createVirtual(taken.name);
      await clickCreate();
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            variant: "error",
            name: taken.name,
            message: "Failed to create channel",
            description: `A channel named "${taken.name}" already exists in folder "${folder.name}". Pick another name.`,
          }),
        ),
      );
      expect(screen.getByPlaceholderText("Channel name")).toBeTruthy();
      const named = await client.channels.retrieve({
        where: { name: { eq: taken.name } },
      });
      expect(named.map(({ key }) => key)).toEqual([taken.key]);
    });

    it("should say where a channel of the taken name sits when it is at the top", async () => {
      const folder = await createFolder();
      const taken = await client.channels.create({
        name: uniqueName("taken"),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      let statuses: Status.NotificationSpec[] = [];
      await openModal({
        params: { group: folder.key },
        onStatuses: (next) => (statuses = next),
      });
      createVirtual(taken.name);
      await clickCreate();
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to create channel",
            description: `A channel named "${taken.name}" already exists at the top. Pick another name.`,
          }),
        ),
      );
    });

    it("should name the folder that holds a folder of the taken name", async () => {
      const folder = await createFolder();
      const sibling = await client.groups.create({
        name: uniqueName("sibling"),
        parent: group.resourceID(folder.key),
      });
      let statuses: Status.NotificationSpec[] = [];
      await openModal({
        params: { group: folder.key },
        onStatuses: (next) => (statuses = next),
      });
      createVirtual(sibling.name);
      await clickCreate();
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to create channel",
            description: `A folder named "${sibling.name}" already exists in folder "${folder.name}". Pick another name.`,
          }),
        ),
      );
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
      const name = uniqueName("noidx");
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
