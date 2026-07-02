// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, device, type ontology } from "@synnaxlabs/client";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import {
  buildResource,
  buildSelection,
  buildState,
} from "@/platform/ontology/testutil";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client = createTestClient();
const TIMEOUT = { timeout: 5000 };

const resourceFor = (key: string, configured: boolean): ontology.Resource =>
  buildResource(device.ontologyID(key), key, { configured });

describe("ConfigureMenuItem", () => {
  describe("guards", () => {
    it("should render nothing when more than one resource is selected", async () => {
      const a = resourceFor("dev-a", false);
      const b = resourceFor("dev-b", false);
      await renderWithConsole(
        <PMenu.Menu>
          <Device.ConfigureMenuItem
            onConfigure={vi.fn()}
            selection={buildSelection({ ids: [a.id, b.id] })}
            state={buildState([a, b])}
          />
        </PMenu.Menu>,
      );
      expect(screen.queryByText("Configure")).toBeNull();
    });

    it("should render nothing when the device is already configured", async () => {
      const r = resourceFor("dev-configured", true);
      await renderWithConsole(
        <PMenu.Menu>
          <Device.ConfigureMenuItem
            onConfigure={vi.fn()}
            selection={buildSelection({ ids: [r.id] })}
            state={buildState([r])}
          />
        </PMenu.Menu>,
      );
      expect(screen.queryByText("Configure")).toBeNull();
    });
  });

  describe("with test client", () => {
    const setup = async (onConfigure = vi.fn()) => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const key = id.create();
      await client.devices.create({
        key,
        name: "Unconfigured Device",
        rack: rack.key,
        location: "loc",
        make: "make",
        model: "model",
        configured: false,
        properties: {},
      });
      const r = resourceFor(key, false);
      const { wrapper: Wrapper } = await createConsoleWrapper({ client });
      const MenuWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <PMenu.Menu>{children}</PMenu.Menu>
        </Wrapper>
      );
      MenuWrapper.displayName = "MenuWrapper";
      render(
        <Device.ConfigureMenuItem
          onConfigure={onConfigure}
          selection={buildSelection({ ids: [r.id] })}
          state={buildState([r])}
        />,
        { wrapper: MenuWrapper },
      );
      return { onConfigure, key };
    };

    it("should render a configure item for a single unconfigured device with permission", async () => {
      await setup();
      await waitFor(() => expect(screen.getByText("Configure")).toBeTruthy(), TIMEOUT);
    });

    it("should invoke onConfigure with the device key when clicked", async () => {
      const { onConfigure, key } = await setup();
      await waitFor(() => expect(screen.getByText("Configure")).toBeTruthy(), TIMEOUT);
      fireEvent.click(screen.getByText("Configure"));
      expect(onConfigure).toHaveBeenCalledWith(key);
    });
  });
});
