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
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CSV } from "@/platform/csv";
import { Modals } from "@/platform/modals";
import { renderWithModals } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const TIME_RANGE = new TimeRange(TimeStamp.seconds(0), TimeStamp.seconds(10));

interface HarnessProps {
  params: CSV.DownloadModalParams;
}

const Harness = ({ params }: HarnessProps): ReactElement => {
  const prompt = CSV.useDownloadModal();
  return (
    <button
      onClick={() => {
        void prompt(params).catch(() => {});
      }}
    >
      open
    </button>
  );
};
Harness.displayName = "Harness";

const findDownloadButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByText("Download")
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error("Download button not found");
  return btn;
};

describe("useDownloadModal", () => {
  describe("form step (no client)", () => {
    const openModal = (params: CSV.DownloadModalParams) => {
      renderWithModals(<Harness params={params} />);
      fireEvent.click(screen.getByRole("button", { name: "open" }));
    };

    it("renders the download prompt with the resource name", async () => {
      openModal({ timeRange: TIME_RANGE, channels: [], name: "My Plot" });
      await waitFor(() =>
        expect(screen.getByText("Download data for My Plot to a CSV")).toBeTruthy(),
      );
    });

    it("renders the channel selector placeholder", async () => {
      openModal({ timeRange: TIME_RANGE, channels: [], name: "My Plot" });
      await waitFor(() =>
        expect(screen.getByText("Select channels to download")).toBeTruthy(),
      );
    });

    it("disables the Download button when no channels are selected", async () => {
      openModal({ timeRange: TIME_RANGE, channels: [], name: "My Plot" });
      await waitFor(() =>
        expect(findDownloadButton().className).toContain("pluto--disabled"),
      );
    });

    it("enables the Download button when channels are preselected", async () => {
      openModal({ timeRange: TIME_RANGE, channels: [1, 2], name: "My Plot" });
      await waitFor(() =>
        expect(findDownloadButton().className).not.toContain("pluto--disabled"),
      );
    });
  });

  describe("with test client", () => {
    let clickedAnchors: HTMLAnchorElement[] = [];
    beforeEach(() => {
      clickedAnchors = [];
      delete (window as unknown as Record<string, unknown>).showSaveFilePicker;
      vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:mock");
      vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
      vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (
        this: HTMLAnchorElement,
      ) {
        clickedAnchors.push(this);
      });
    });
    afterEach(() => {
      vi.restoreAllMocks();
    });

    it("reads the selected channels and downloads a CSV", async () => {
      const client = createTestClient();
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const ModalsWrapper: FC<PropsWithChildren> = ({ children }) => (
        <Wrapper>
          <Session.Modals.Provider>{children}</Session.Modals.Provider>
        </Wrapper>
      );
      ModalsWrapper.displayName = "ModalsWrapper";

      const indexCh = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: indexCh.key,
      });
      const timestamps = Array.from({ length: 5 }, (_, i) => TimeStamp.seconds(i + 1));
      await client.write(TimeStamp.seconds(1), {
        [indexCh.key]: timestamps,
        [dataCh.key]: [1, 2, 3, 4, 5],
      });

      render(
        <>
          <Harness
            params={{
              timeRange: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(6)),
              channels: [dataCh.key],
              name: "export",
            }}
          />
          <Modals.Stack />
        </>,
        { wrapper: ModalsWrapper },
      );
      fireEvent.click(screen.getByRole("button", { name: "open" }));

      const downloadButton = await waitFor(() => {
        const btn = findDownloadButton();
        expect(btn.className).not.toContain("pluto--disabled");
        return btn;
      });
      await act(async () => {
        fireEvent.click(downloadButton);
      });

      await waitFor(() => expect(clickedAnchors).toHaveLength(1), { timeout: 5000 });
      expect(clickedAnchors[0].download).toEqual("export.csv");
    });
  });
});
