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
import { Task } from "@/platform/task";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client = createTestClient();
const TIMEOUT = { timeout: 5000 };

const configs: Device.TaskContextMenuItemConfig[] = [
  {
    itemKey: "read",
    label: "Create Read Task",
    layout: { ...Task.LAYOUT, type: "test_read", key: "test_read" },
  },
];

const resourceFor = (key: string, configured: boolean): ontology.Resource =>
  buildResource(device.ontologyID(key), key, { configured });

describe("TaskContextMenuItems", () => {
  it("should render nothing without task-create permission", async () => {
    const r = resourceFor("dev-1", true);
    await renderWithConsole(
      <PMenu.Menu>
        <Device.TaskContextMenuItems
          onConfigure={vi.fn()}
          selection={buildSelection({ ids: [r.id] })}
          state={buildState([r])}
          taskContextMenuItemConfigs={configs}
        />
      </PMenu.Menu>,
    );
    expect(screen.queryByText("Create Read Task")).toBeNull();
  });

  describe("with test client", () => {
    const setup = async (configured: boolean, onConfigure = vi.fn()) => {
      const r = resourceFor(id.create(), configured);
      const { wrapper: Wrapper, store } = await createConsoleWrapper({ client });
      const MenuWrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <PMenu.Menu>{children}</PMenu.Menu>
        </Wrapper>
      );
      MenuWrapper.displayName = "MenuWrapper";
      render(
        <Device.TaskContextMenuItems
          onConfigure={onConfigure}
          selection={buildSelection({ ids: [r.id] })}
          state={buildState([r])}
          taskContextMenuItemConfigs={configs}
        />,
        { wrapper: MenuWrapper },
      );
      return { onConfigure, key: r.id.key, store };
    };

    it("should render a menu item per config when create is granted", async () => {
      await setup(true);
      await waitFor(
        () => expect(screen.getByText("Create Read Task")).toBeTruthy(),
        TIMEOUT,
      );
    });

    it("should configure the device before placing the layout when unconfigured", async () => {
      const { onConfigure, key } = await setup(false);
      await waitFor(
        () => expect(screen.getByText("Create Read Task")).toBeTruthy(),
        TIMEOUT,
      );
      fireEvent.click(screen.getByText("Create Read Task"));
      expect(onConfigure).toHaveBeenCalledWith(key);
    });

    it("should not configure the device when already configured", async () => {
      const { onConfigure } = await setup(true);
      await waitFor(
        () => expect(screen.getByText("Create Read Task")).toBeTruthy(),
        TIMEOUT,
      );
      fireEvent.click(screen.getByText("Create Read Task"));
      expect(onConfigure).not.toHaveBeenCalled();
    });
  });
});
