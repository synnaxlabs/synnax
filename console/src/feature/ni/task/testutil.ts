// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { expect } from "vitest";

import * as Device from "@/feature/ni/device/types";
import { type Panel } from "@/platform/panel";
import {
  renderTaskFormTab,
  type RenderTaskFormViewOptions,
  type RenderTaskFormViewResult,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

export interface CreateNIDeviceOptions extends Partial<Omit<Device.New, "properties">> {
  properties?: Partial<Device.Properties>;
}

/**
 * Creates a rack and a configured NI device on the live cluster. The device gets a
 * unique identifier so channels created from it never collide across runs.
 */
export const createNIDevice = async (
  client: Synnax,
  { properties, ...overrides }: CreateNIDeviceOptions = {},
): Promise<Device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const dev = await client.devices.create(
    {
      key: id.create(),
      name: uniqueName("ni_dev"),
      rack: rack.key,
      location: "Dev1",
      make: Device.MAKE,
      model: "TEST-MODEL",
      configured: true,
      properties: { ...Device.ZERO_PROPERTIES, identifier: id.create(), ...properties },
      ...overrides,
    },
    Device.SCHEMAS,
  );
  // Cluster metadata is eventually consistent; wait until the device is retrievable
  // so task configuration flows see it.
  await waitFor(async () => {
    await client.devices.retrieve({ key: dev.key, schemas: Device.SCHEMAS });
  });
  return dev;
};

export interface RenderNITaskFormResult extends RenderTaskFormViewResult {
  /** Live view of the status notifications raised while the form is mounted. */
  statuses: Status.NotificationSpec[];
}

/**
 * Renders a wrapped NI task form the way the mosaic does (via renderTaskFormTab)
 * with a status capture mounted alongside it, so specs can assert on notifications
 * raised by the configure flow.
 */
export const renderNITaskForm = async (
  Tab: Panel.Tab,
  type: string,
  options: Omit<RenderTaskFormViewOptions, "onStatuses"> = {},
): Promise<RenderNITaskFormResult> => {
  const statuses: Status.NotificationSpec[] = [];
  const result = await renderTaskFormTab(Tab, type, {
    ...options,
    onStatuses: (next) => {
      statuses.length = 0;
      statuses.push(...next);
    },
  });
  return { ...result, statuses };
};

/**
 * Polls the captured statuses until one's message or description matches. Uses
 * expect.poll with an extended timeout because the failing operation round-trips
 * against the live cluster before raising the status.
 */
export const awaitStatusDescription = async (
  statuses: Status.NotificationSpec[],
  pattern: RegExp,
): Promise<void> => {
  await expect
    .poll(
      () =>
        statuses.some(
          (s) => pattern.test(s.description ?? "") || pattern.test(s.message),
        ),
      { timeout: 10_000, message: `no status matching ${pattern}` },
    )
    .toBe(true);
};
