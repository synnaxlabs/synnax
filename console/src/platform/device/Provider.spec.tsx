// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Device } from "@/platform/device";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client = createTestClient();
const TIMEOUT = { timeout: 5000 };

const schema = z.object({ config: z.object({ device: z.string() }) });

const FormFixture = ({
  deviceKey,
  children,
}: PropsWithChildren<{ deviceKey: string }>): ReactElement => {
  const methods = Form.use({ schema, values: { config: { device: deviceKey } } });
  return <Form.Form<typeof schema> {...methods}>{children}</Form.Form>;
};
FormFixture.displayName = "FormFixture";

const renderProvider = (deviceKey: string, node: ReactElement) =>
  renderWithConsole(<FormFixture deviceKey={deviceKey}>{node}</FormFixture>);

describe("device Provider", () => {
  describe("no device selected", () => {
    it("should render the default none-selected content when no device is set", async () => {
      await renderProvider(
        "",
        <Device.Provider canConfigure onConfigure={vi.fn()}>
          {({ device }) => <div>{device.name}</div>}
        </Device.Provider>,
      );
      expect(screen.getByText("No device selected.")).toBeTruthy();
    });

    it("should render custom none-selected content when provided", async () => {
      await renderProvider(
        "",
        <Device.Provider
          canConfigure
          onConfigure={vi.fn()}
          noneSelectedContent={<div>pick something</div>}
        >
          {({ device }) => <div>{device.name}</div>}
        </Device.Provider>,
      );
      expect(screen.getByText("pick something")).toBeTruthy();
    });
  });

  describe("with test client", () => {
    const createDevice = async (configured: boolean) => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const key = id.create();
      await client.devices.create({
        key,
        name: configured ? "Configured Device" : "Unconfigured Device",
        rack: rack.key,
        location: "loc",
        make: "make",
        model: "model",
        configured,
        properties: {},
      });
      return key;
    };

    const renderWithDevice = async (key: string, node: ReactElement) => {
      const { wrapper: Wrapper } = await createConsoleWrapper({ client });
      const FormWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <FormFixture deviceKey={key}>{children}</FormFixture>
        </Wrapper>
      );
      FormWrapper.displayName = "FormWrapper";
      render(node, { wrapper: FormWrapper });
    };

    it("should render the children with the device once configured", async () => {
      const key = await createDevice(true);
      await renderWithDevice(
        key,
        <Device.Provider canConfigure onConfigure={vi.fn()}>
          {({ device }) => <div>rendered {device.name}</div>}
        </Device.Provider>,
      );
      await waitFor(
        () => expect(screen.getByText("rendered Configured Device")).toBeTruthy(),
        TIMEOUT,
      );
    });

    it("should prompt to configure an unconfigured device", async () => {
      const key = await createDevice(false);
      const onConfigure = vi.fn();
      await renderWithDevice(
        key,
        <Device.Provider canConfigure onConfigure={onConfigure}>
          {({ device }) => <div>rendered {device.name}</div>}
        </Device.Provider>,
      );
      await waitFor(
        () => expect(screen.getByText("Configure Unconfigured Device")).toBeTruthy(),
        TIMEOUT,
      );
      fireEvent.click(screen.getByText("Configure Unconfigured Device"));
      expect(onConfigure).toHaveBeenCalledWith(key);
    });

    it("should hide the configure action when configuration is not allowed", async () => {
      const key = await createDevice(false);
      await renderWithDevice(
        key,
        <Device.Provider canConfigure={false} onConfigure={vi.fn()}>
          {({ device }) => <div>rendered {device.name}</div>}
        </Device.Provider>,
      );
      await waitFor(
        () =>
          expect(
            screen.getByText("Unconfigured Device is not configured."),
          ).toBeTruthy(),
        TIMEOUT,
      );
      expect(screen.queryByText("Configure Unconfigured Device")).toBeNull();
    });
  });
});
