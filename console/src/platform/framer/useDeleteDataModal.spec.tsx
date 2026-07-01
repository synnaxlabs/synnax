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
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { useDeleteDataModal } from "@/platform/framer/useDeleteDataModal";
import { Modals } from "@/platform/modals";
import { renderWithModals } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const Harness = (): ReactElement => {
  const open = useDeleteDataModal();
  return <button onClick={() => open()}>open</button>;
};
Harness.displayName = "Harness";

const openModal = (): RenderResult => {
  const result = renderWithModals(<Harness />);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return result;
};

const getNextButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByText("Next")
    .find((el) => el.closest("button"))
    ?.closest("button");
  if (btn == null) throw new Error("Next button not found");
  return btn;
};

const getCheckboxes = (): NodeListOf<HTMLInputElement> =>
  document.body.querySelectorAll<HTMLInputElement>("input[type='checkbox']");

describe("DeleteModal", () => {
  describe("form step", () => {
    it("should render the form step with the title", async () => {
      openModal();
      await waitFor(() => expect(screen.getByText("Delete Data")).toBeTruthy());
    });

    it("should render the channel selector placeholder", async () => {
      openModal();
      await waitFor(() =>
        expect(screen.getByText("Select channels to delete")).toBeTruthy(),
      );
    });

    it("should render the Next button", async () => {
      openModal();
      await waitFor(() => expect(getNextButton()).toBeTruthy());
    });

    it("should disable the Next button when no channels are selected", async () => {
      openModal();
      await waitFor(() =>
        expect(getNextButton().className).toContain("pluto--disabled"),
      );
    });

    it("should have both time checkboxes checked by default", async () => {
      openModal();
      await waitFor(() =>
        expect(screen.getByText("From beginning of time")).toBeTruthy(),
      );
      expect(screen.getByText("To end of time")).toBeTruthy();
      getCheckboxes().forEach((cb) => {
        expect(cb.checked).toBe(true);
      });
    });

    it("should not show DateTime inputs when both checkboxes are checked", async () => {
      openModal();
      await waitFor(() =>
        expect(screen.getByText("From beginning of time")).toBeTruthy(),
      );
      expect(screen.queryByText("From")).toBeNull();
      expect(screen.queryByText("To")).toBeNull();
    });

    it("should show From DateTime input when the start checkbox is unchecked", async () => {
      openModal();
      await waitFor(() => expect(getCheckboxes().length).toBeGreaterThan(0));
      fireEvent.click(getCheckboxes()[0]);
      expect(screen.getByText("From")).toBeTruthy();
    });

    it("should show To DateTime input when the end checkbox is unchecked", async () => {
      openModal();
      await waitFor(() => expect(getCheckboxes().length).toBeGreaterThan(0));
      fireEvent.click(getCheckboxes()[1]);
      expect(screen.getByText("To")).toBeTruthy();
    });
  });

  describe("with test client", () => {
    it("should delete data via the modal flow", async () => {
      const client = createTestClient();
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const ModalsWrapper: FC<PropsWithChildren> = ({ children }) => (
        <Wrapper>
          <Session.Modals.Provider>{children}</Session.Modals.Provider>
        </Wrapper>
      );
      ModalsWrapper.displayName = "ModalsWrapper";

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

      render(
        <>
          <Harness />
          <Modals.Stack />
        </>,
        { wrapper: ModalsWrapper },
      );
      fireEvent.click(screen.getByRole("button", { name: "open" }));

      fireEvent.click(await screen.findByText("Select channels to delete"));

      await waitFor(() => {
        expect(screen.getByText(dataCh.name)).toBeTruthy();
      });
      fireEvent.click(screen.getByText(dataCh.name));

      const nextButton = getNextButton();
      expect(nextButton.className).not.toContain("pluto--disabled");
      await act(async () => {
        fireEvent.click(nextButton);
      });

      await waitFor(() => {
        expect(
          screen.getByText("Are you sure you want to delete this data?"),
        ).toBeTruthy();
      });
      expect(screen.getByText("This action is irreversible.")).toBeTruthy();

      const deleteButton = screen
        .getAllByText("Delete")
        .map((el) => el.closest<HTMLButtonElement>("button"))
        .find((b) => b != null);
      if (deleteButton == null) throw new Error("Delete button not found");
      await act(async () => {
        fireEvent.click(deleteButton);
      });

      await waitFor(() =>
        expect(
          screen.queryByText("Are you sure you want to delete this data?"),
        ).toBeNull(),
      );

      await waitFor(async () => {
        const after = await client.read(TimeRange.MAX, dataCh.key);
        expect(after.length).toEqual(0);
      });
    });
  });
});
