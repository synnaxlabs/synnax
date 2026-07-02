// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client = createTestClient();
const TIMEOUT = { timeout: 5000 };

const useConfigureModal = Modals.create<Device.ConfigureParams>(
  ({ deviceKey, close }) => (
    <Device.Configure
      deviceKey={deviceKey}
      close={close}
      icon={<Icon.Hardware />}
      initialProperties={{}}
    />
  ),
);

describe("device Configure", () => {
  it("should not render the form while the device has not been retrieved", async () => {
    await renderWithConsole(
      <Device.Configure
        deviceKey={id.create()}
        close={vi.fn()}
        icon={<Icon.Hardware />}
        initialProperties={{}}
      />,
    );
    expect(screen.queryByText(/enter a name so it's easy to look up later/)).toBeNull();
  });

  describe("with test client", () => {
    const Harness = ({ deviceKey }: { deviceKey: string }): ReactElement => {
      const open = useConfigureModal();
      return <button onClick={() => open({ deviceKey })}>open</button>;
    };
    Harness.displayName = "Harness";

    const setup = async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const key = id.create();
      await client.devices.create({
        key,
        name: "My Sensor",
        rack: rack.key,
        location: "loc",
        make: "make",
        model: "model",
        configured: false,
        properties: {},
      });
      const { wrapper: Wrapper } = await createConsoleWrapper({ client });
      const ModalWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>{children}</Wrapper>
      );
      ModalWrapper.displayName = "ModalWrapper";
      render(
        <>
          <Harness deviceKey={key} />
          <Modals.Stack />
        </>,
        { wrapper: ModalWrapper },
      );
      fireEvent.click(screen.getByRole("button", { name: "open" }));
      return { key };
    };

    it("should render the name step prefilled with the device name", async () => {
      await setup();
      await waitFor(
        () => expect(screen.getByText(/enter a name/)).toBeTruthy(),
        TIMEOUT,
      );
      const nameInput = screen.getByRole<HTMLInputElement>("textbox");
      expect(nameInput.value).toEqual("My Sensor");
    });

    it("should advance to the identifier step and configure the device", async () => {
      const { key } = await setup();
      await waitFor(
        () => expect(screen.getByText(/enter a name/)).toBeTruthy(),
        TIMEOUT,
      );
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Next" }));
      });
      await waitFor(
        () => expect(screen.getByText(/short identifier/)).toBeTruthy(),
        TIMEOUT,
      );
      const identifierInput = screen.getByRole<HTMLInputElement>("textbox");
      fireEvent.change(identifierInput, { target: { value: "sensor_1" } });
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: "Save" }));
      });
      await waitFor(async () => {
        const updated = await client.devices.retrieve({ key });
        expect(updated.configured).toBe(true);
        expect(updated.properties.identifier).toEqual("sensor_1");
      }, TIMEOUT);
    });
  });
});
