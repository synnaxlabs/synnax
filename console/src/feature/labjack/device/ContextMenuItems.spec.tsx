// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { createDeviceResource, createTestDevice } from "@/platform/device/testutil";
import { Errors } from "@/platform/errors";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderSuspended,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const renderItems = async () => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  store.dispatch(Session.Project.select(proj.key));
  const dev = await createTestDevice(client, { name: uniqueName("lj_dev") });
  const resource = createDeviceResource({ ...dev, configured: true });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: [resource.id] }),
    state: createState([resource]),
  };
  await renderSuspended(
    <PMenu.Menu>
      <Errors.SuspenseBoundary loading={null}>
        <LabJack.Device.ContextMenuItems {...props} />
      </Errors.SuspenseBoundary>
    </PMenu.Menu>,
    { wrapper },
  );
  return { store, key: dev.key };
};

describe("LabJack device ContextMenuItems", () => {
  it("should open the read task view bound to the device", async () => {
    const { store, key } = await renderItems();
    fireEvent.click(await screen.findByText("Create read task"));
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: LabJack.Task.READ_TYPE,
      args: { deviceKey: key },
    });
  });

  it("should open the write task view bound to the device", async () => {
    const { store, key } = await renderItems();
    fireEvent.click(await screen.findByText("Create write task"));
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: LabJack.Task.WRITE_TYPE,
      args: { deviceKey: key },
    });
  });
});
