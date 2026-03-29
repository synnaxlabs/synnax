// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DataType } from "@synnaxlabs/client";
import { id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DeleteModal } from "@/framer/DeleteModal";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

const NullWrapper = createSynnaxWrapper({ client: null });

const renderModal = (
  Wrapper: FC<PropsWithChildren> = NullWrapper,
  onClose = vi.fn(),
) => ({
  result: render(
    <Wrapper>
      <DeleteModal layoutKey="test" onClose={onClose} visible focused />
    </Wrapper>,
  ),
  onClose,
});

const getNextButton = (c: ReturnType<typeof render>): HTMLButtonElement => {
  const buttons = c.getAllByText("Next");
  const btn = buttons.find((el) => el.closest("button"))?.closest("button");
  if (btn == null) throw new Error("Next button not found");
  return btn;
};

describe("DeleteModal", () => {
  describe("form step", () => {
    it("should render the form step with the title", () => {
      const { result: c } = renderModal();
      expect(c.getByText("Delete Data")).toBeTruthy();
    });

    it("should render the channel selector placeholder", () => {
      const { result: c } = renderModal();
      expect(c.getByText("Select channels to delete")).toBeTruthy();
    });

    it("should render the Next button", () => {
      const { result: c } = renderModal();
      expect(getNextButton(c)).toBeTruthy();
    });

    it("should disable the Next button when no channels are selected", () => {
      const { result: c } = renderModal();
      expect(getNextButton(c).className).toContain("pluto--disabled");
    });

    it("should have both time checkboxes checked by default", () => {
      const { result: c } = renderModal();
      expect(c.getByText("From beginning of time")).toBeTruthy();
      expect(c.getByText("To end of time")).toBeTruthy();
      const checkboxes = c.container.querySelectorAll<HTMLInputElement>(
        "input[type='checkbox']",
      );
      checkboxes.forEach((cb) => {
        expect(cb.checked).toBe(true);
      });
    });

    it("should not show DateTime inputs when both checkboxes are checked", () => {
      const { result: c } = renderModal();
      expect(c.queryByText("From")).toBeNull();
      expect(c.queryByText("To")).toBeNull();
    });

    it("should show From DateTime input when the start checkbox is unchecked", () => {
      const { result: c } = renderModal();
      const checkboxes = c.container.querySelectorAll<HTMLInputElement>(
        "input[type='checkbox']",
      );
      const startCheckbox = checkboxes[0];
      fireEvent.click(startCheckbox);
      expect(c.getByText("From")).toBeTruthy();
    });

    it("should show To DateTime input when the end checkbox is unchecked", () => {
      const { result: c } = renderModal();
      const checkboxes = c.container.querySelectorAll<HTMLInputElement>(
        "input[type='checkbox']",
      );
      const endCheckbox = checkboxes[1];
      fireEvent.click(endCheckbox);
      expect(c.getByText("To")).toBeTruthy();
    });
  });

  describe("with test client", () => {
    const client = createTestClient();
    let wrapper: FC<PropsWithChildren>;

    beforeEach(async () => {
      wrapper = await createAsyncSynnaxWrapper({ client });
    });

    it("should delete data via the modal flow", async () => {
      const indexCh = await client.channels.create({
        leaseholder: 1,
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        leaseholder: 1,
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: indexCh.key,
      });

      const timestamps = Array.from({ length: 10 }, (_, i) => TimeStamp.seconds(i + 1));
      await client.write(TimeStamp.seconds(1), {
        [indexCh.key]: timestamps,
        [dataCh.key]: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10],
      });

      const before = await client.read(TimeRange.MAX, dataCh.key);
      expect(before.data.length).toEqual(10);

      const { result: c, onClose } = renderModal(wrapper);

      // Open the channel selector dropdown
      fireEvent.click(c.getByText("Select channels to delete"));

      // Wait for channels to load from the server and select the data channel
      await waitFor(() => {
        expect(c.getByText(dataCh.name)).toBeTruthy();
      });
      fireEvent.click(c.getByText(dataCh.name));

      // Next button should now be enabled — click it
      const nextButton = getNextButton(c);
      expect(nextButton.className).not.toContain("pluto--disabled");
      await act(async () => {
        fireEvent.click(nextButton);
      });

      // Should be on the confirm step
      await waitFor(() => {
        expect(c.getByText("Are you sure you want to delete this data?")).toBeTruthy();
      });
      expect(c.getByText("This action is irreversible.")).toBeTruthy();

      // Press and hold the Delete button past the onClickDelay
      const deleteButton = c.getByText("Delete").closest("button") as HTMLButtonElement;
      vi.useFakeTimers();
      await act(async () => {
        fireEvent.mouseDown(deleteButton);
        vi.advanceTimersByTime(1000);
      });
      vi.useRealTimers();

      // Modal should have closed
      expect(onClose).toHaveBeenCalled();

      // Verify data was actually deleted on the server
      await waitFor(async () => {
        const after = await client.read(TimeRange.MAX, dataCh.key);
        expect(after.length).toEqual(0);
      });
    });
  });
});
