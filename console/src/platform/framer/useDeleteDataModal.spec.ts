// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type Synnax } from "@synnaxlabs/client";
import { TimeRange, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createTestIndexedPair,
  type TestIndexedPair,
} from "@/platform/channel/testutil";
import { Framer } from "@/platform/framer";
import {
  findButton,
  getSwitch,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { stubGeometry } from "@/testutil";

const client = createTestClient();

const SAMPLES = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

const openModal = async (c: Synnax | null = null): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Framer.useDeleteDataModal, [], { client: c });
  await screen.findByText("Delete Data");
  return handle;
};

const createWrittenPair = async (): Promise<TestIndexedPair> => {
  const pair = await createTestIndexedPair(client);
  const timestamps = SAMPLES.map((i) => TimeStamp.seconds(i));
  await client.write(TimeStamp.seconds(1), {
    [pair.index.key]: timestamps,
    [pair.data.key]: SAMPLES,
  });
  return pair;
};

const selectChannel = async (name: string): Promise<void> => {
  fireEvent.click(screen.getByText("Select channels to delete"));
  fireEvent.change(await screen.findByPlaceholderText("Search channels...", {}), {
    target: { value: name },
  });
  fireEvent.click(await screen.findByText(name, {}));
};

const goToConfirmStep = async (): Promise<void> => {
  const next = findButton("Next");
  expect(next.className).not.toContain("pluto--disabled");
  await act(async () => {
    fireEvent.click(next);
  });
  await screen.findByText("Are you sure you want to delete this data?");
};

stubGeometry();

describe("DeleteModal", () => {
  describe("form step", () => {
    it("should disable the Next button while no channels are selected", async () => {
      await openModal();
      await waitFor(() =>
        expect(findButton("Next").className).toContain("pluto--disabled"),
      );
    });

    it("should default to the full time range with no explicit bounds", async () => {
      await openModal();
      expect(getSwitch("From beginning of time").checked).toBe(true);
      expect(getSwitch("To end of time").checked).toBe(true);
      expect(screen.queryByText("From")).toBeNull();
      expect(screen.queryByText("To")).toBeNull();
    });

    it("should reveal the bound inputs when the checkboxes are unchecked", async () => {
      await openModal();
      fireEvent.click(getSwitch("From beginning of time"));
      expect(screen.getByText("From")).toBeTruthy();
      fireEvent.click(getSwitch("To end of time"));
      expect(screen.getByText("To")).toBeTruthy();
    });
  });

  describe("with test client", () => {
    it("should delete the channel's data through the confirm flow", async () => {
      const { data } = await createWrittenPair();
      const before = await client.read(TimeRange.MAX, data.key);
      expect(before.data.length).toEqual(SAMPLES.length);

      await openModal(client);
      await selectChannel(data.name);
      await goToConfirmStep();

      await act(async () => {
        fireEvent.click(findButton("Delete"));
      });
      await waitFor(() =>
        expect(
          screen.queryByText("Are you sure you want to delete this data?"),
        ).toBeNull(),
      );
      await waitFor(async () => {
        const after = await client.read(TimeRange.MAX, data.key);
        expect(after.length).toEqual(0);
      });
    });

    it("should preserve the selection and delete nothing when backing out of the confirmation", async () => {
      const { data } = await createWrittenPair();
      await openModal(client);
      await selectChannel(data.name);
      await goToConfirmStep();

      await act(async () => {
        fireEvent.click(findButton("Back"));
      });
      await screen.findByText("Delete Data");
      expect(screen.getByText(data.name)).toBeTruthy();
      expect(findButton("Next").className).not.toContain("pluto--disabled");

      const after = await client.read(TimeRange.MAX, data.key);
      expect(after.data.length).toEqual(SAMPLES.length);
    });
  });
});
