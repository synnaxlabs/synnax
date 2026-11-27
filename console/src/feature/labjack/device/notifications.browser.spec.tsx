// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { Modals } from "@/platform/modals";
import { Notifications } from "@/platform/notifications";
import { createConsoleWrapper, installPortalRoot } from "@/testutil";

const client = createTestClient();

const Harness = ({ details }: { details: unknown }): ReactElement => {
  const add = Status.useAdder();
  return (
    <>
      <button
        onClick={() =>
          add({ variant: "info", message: "LabJack T7 connected", details })
        }
      >
        add
      </button>
      <Notifications.Feed notifications={LabJack.Device.NOTIFICATIONS} />
      <Modals.Stack />
    </>
  );
};

const addStatus = async (details: unknown): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Harness details={details} />, { wrapper });
  fireEvent.click(screen.getByText("add"));
};

describe("LabJack.Device.NOTIFICATIONS", () => {
  installPortalRoot();

  it("should open the typed configure form from Configure", async () => {
    const rack = await client.racks.create({ name: uniqueName("labjack_rack") });
    const dev = await client.labjack.devices.create({
      rack: rack.key,
      name: uniqueName("labjack_dev"),
      serial: Math.floor(Math.random() * 400_000) * 10_000 + 6,
      connection: "LJM_ctUSB",
      model: "t7",
      ports: { model: "t7" },
    });
    await addStatus({ unconfiguredLabJackDevice: dev.key });
    fireEvent.click(await screen.findByText("Configure"));
    expect(await screen.findByDisplayValue(dev.name)).toBeTruthy();
  });

  it("should not match a status without a LabJack device key", async () => {
    await addStatus({ other: "value" });
    expect(await screen.findByText("LabJack T7 connected")).toBeTruthy();
    expect(screen.queryByText("Configure")).toBeNull();
  });
});
