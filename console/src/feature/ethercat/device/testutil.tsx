// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { act, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { vi } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { type Device } from "@/platform/device";
import { createDeviceResource } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

interface SlaveLoaderProps {
  deviceKey: device.Key;
}

const SlaveLoader = ({ deviceKey }: SlaveLoaderProps): ReactElement | null => {
  const { data } = EtherCAT.Device.useResultSlave({ key: deviceKey });
  return data == null ? null : <span>{`loaded:${deviceKey}`}</span>;
};

/**
 * Mounts the context menu in two phases: first loads the slaves into the flux device
 * cache (the tree does this in production before a menu can open), then mounts the
 * menu items so their enabled/disabled computation sees the cached devices.
 */
export const renderContextMenu = async (
  client: Synnax,
  devices: EtherCAT.Device.SlaveDevice[],
) => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
  });
  store.dispatch(Session.Project.select(proj.key));
  const keys = devices.map((d) => d.key);
  const loaders = keys.map((k) => <SlaveLoader key={k} deviceKey={k} />);
  const result = render(<div>{loaders}</div>, { wrapper });
  for (const k of keys) await screen.findByText(`loaded:${k}`);
  const resources = devices.map((d) =>
    createDeviceResource({ key: d.key, name: d.name, configured: d.configured }),
  );
  const props: Device.ContextMenuItemsProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: resources.map((r) => r.id) }),
    state: createState(resources),
    onConfigure: vi.fn(),
  };
  // The menu suspends on the slaves fetch; a tree that suspends inside a sync
  // act never commits, so the mount needs an async act (see renderHookSuspended).
  await act(async () => {
    result.rerender(
      <Triggers.Provider>
        {loaders}
        <Menu.Menu>
          <EtherCAT.Device.ContextMenuItems {...props} />
        </Menu.Menu>
        <Modals.Stack />
      </Triggers.Provider>,
    );
  });
  return { store };
};

/** Creates a slave with the properties on a new rack. */
export const createSlave = async (
  client: Synnax,
  enabled: boolean,
  properties: Partial<EtherCAT.Device.SlaveProperties> = {},
) => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  return await createSlaveDevice(client, rack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    enabled,
    ...properties,
  });
};
