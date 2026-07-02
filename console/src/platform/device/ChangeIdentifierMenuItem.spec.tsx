// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, device, type ontology } from "@synnaxlabs/client";
import { Menu as PMenu, type Status } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";
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

// Runs the async callback the menu item hands to handleError, as the real handler does.
const runHandler: Status.ErrorHandler = (funcOrExc) => {
  if (typeof funcOrExc === "function") void funcOrExc();
};

describe("ChangeIdentifierMenuItem", () => {
  describe("guards", () => {
    it("should render nothing when more than one resource is selected", async () => {
      const a = resourceFor("dev-a", true);
      const b = resourceFor("dev-b", true);
      await renderWithConsole(
        <PMenu.Menu>
          <Device.ChangeIdentifierMenuItem
            icon="Hardware"
            selection={buildSelection({ ids: [a.id, b.id] })}
            state={buildState([a, b])}
            handleError={() => {}}
          />
        </PMenu.Menu>,
      );
      expect(screen.queryByText("Change identifier")).toBeNull();
    });

    it("should render nothing when the device is not configured", async () => {
      const r = resourceFor("dev-unconfigured", false);
      await renderWithConsole(
        <PMenu.Menu>
          <Device.ChangeIdentifierMenuItem
            icon="Hardware"
            selection={buildSelection({ ids: [r.id] })}
            state={buildState([r])}
            handleError={() => {}}
          />
        </PMenu.Menu>,
      );
      expect(screen.queryByText("Change identifier")).toBeNull();
    });
  });

  describe("with test client", () => {
    const setup = async () => {
      const rack = await client.racks.create({ name: `rack-${id.create()}` });
      const key = id.create();
      await client.devices.create({
        key,
        name: "Configured Device",
        rack: rack.key,
        location: "loc",
        make: "make",
        model: "model",
        configured: true,
        properties: { identifier: "old-id" },
      });
      const r = resourceFor(key, true);
      const { wrapper: Wrapper } = await createConsoleWrapper({ client });
      const MenuWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <PMenu.Menu>{children}</PMenu.Menu>
          <Modals.Stack />
        </Wrapper>
      );
      MenuWrapper.displayName = "MenuWrapper";
      render(
        <Device.ChangeIdentifierMenuItem
          icon="Hardware"
          selection={buildSelection({ ids: [r.id] })}
          state={buildState([r])}
          handleError={runHandler}
        />,
        { wrapper: MenuWrapper },
      );
      return { key };
    };

    it("should render the change-identifier item for a configured device", async () => {
      await setup();
      await waitFor(
        () => expect(screen.getByText("Change identifier")).toBeTruthy(),
        TIMEOUT,
      );
    });

    it("should update the device identifier through the rename modal", async () => {
      const { key } = await setup();
      await waitFor(
        () => expect(screen.getByText("Change identifier")).toBeTruthy(),
        TIMEOUT,
      );
      fireEvent.click(screen.getByText("Change identifier"));
      const input = await screen.findByRole("textbox");
      await waitFor(() => expect((input as HTMLInputElement).value).toEqual("old-id"));
      fireEvent.change(input, { target: { value: "new-id" } });
      fireEvent.click(screen.getByRole("button", { name: "Save" }));
      await waitFor(async () => {
        const updated = await client.devices.retrieve({ key });
        expect(updated.properties.identifier).toEqual("new-id");
      }, TIMEOUT);
    });
  });
});
