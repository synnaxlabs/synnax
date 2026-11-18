// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { Device } from "@/platform/device";
import { createWarningScanner } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand } from "@/platform/task/testutil";
import { CaptureStatuses, uniqueName } from "@/testutil";

const client = createTestClient();

const BLOCKED = "No EtherCAT modules found on Ethernet 2.";

/**
 * Opens the find view on a rack whose EtherCAT scanner warns that the DewesoftX filter
 * blocks an adapter, with the data of the warning.
 */
const renderFind = async (
  data: unknown,
): Promise<{ scanner: task.Task; statuses: () => Status.NotificationSpec[] }> => {
  const hardware = { ...EtherCAT.Device.HARDWARE, key: uniqueName("integration") };
  const scanner = await createWarningScanner(client, hardware, {
    description: BLOCKED,
    data,
  });
  let statuses: Status.NotificationSpec[] = [];
  await renderModalOpener(Device.useFindModal, [{ hardware }], {
    client,
    extra: createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
  });
  await screen.findByText(BLOCKED);
  return { scanner, statuses: () => statuses };
};

describe("EtherCAT scan action", () => {
  it("should turn off the DewesoftX filter once the user confirms", async () => {
    const { scanner, statuses } = await renderFind({ filtered: ["\\Device\\NPF_{0}"] });
    const next = "If Windows asks for administrator access, click Yes.";
    const driver = await answerNextCommand(client, scanner, {
      variant: "success",
      message: "Started to turn off the DewesoftX filter",
      description: next,
    });
    fireEvent.click(findButton("Turn off DewesoftX filter"));
    fireEvent.click(await screen.findByRole("button", { name: "Turn off" }));
    expect((await driver.answered).type).toBe("disable_dewesoft_filter");
    await waitFor(() =>
      expect(
        statuses().find(
          (s) =>
            s.variant === "success" &&
            s.message === "Started to turn off the DewesoftX filter" &&
            s.description === next,
        ),
      ).toBeDefined(),
    );
  });

  it("should show why the Driver could not turn off the filter", async () => {
    const { scanner, statuses } = await renderFind({ filtered: ["\\Device\\NPF_{0}"] });
    const reason = "Only a Driver on Windows can turn off the DewesoftX filter.";
    const driver = await answerNextCommand(client, scanner, {
      variant: "error",
      message: "Failed to turn off the DewesoftX filter",
      description: reason,
    });
    fireEvent.click(findButton("Turn off DewesoftX filter"));
    fireEvent.click(await screen.findByRole("button", { name: "Turn off" }));
    await driver.answered;
    await waitFor(() =>
      expect(
        statuses().find(
          (s) =>
            s.message === "Failed to turn off the DewesoftX filter" &&
            s.description === reason,
        ),
      ).toBeDefined(),
    );
  });

  it("should send no command when the user cancels", async () => {
    await renderFind({ filtered: ["\\Device\\NPF_{0}"] });
    fireEvent.click(findButton("Turn off DewesoftX filter"));
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await act(async () => {});
    // A sent command holds the button in its loading state, since no Driver answers.
    expect(
      findButton("Turn off DewesoftX filter").getAttribute("aria-disabled"),
    ).toBeNull();
  });

  it("should not offer to turn off the filter when it blocks no adapter", async () => {
    await renderFind(undefined);
    expect(screen.queryByText("Turn off DewesoftX filter")).toBeNull();
  });
});
