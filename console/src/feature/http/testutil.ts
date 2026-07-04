// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";

import { MAKE, type Properties, ZERO_PROPERTIES } from "@/feature/http/device/types";
import { createTestDevice } from "@/platform/device/testutil";
import { getBySelector, getIconButton, getInputItem, uniqueName } from "@/testutil";

export interface CreateHTTPDeviceOptions {
  configured?: boolean;
  properties?: Partial<Properties>;
}

/** Creates a rack and a configured HTTP server device on the live cluster. */
export const createHTTPDevice = async (
  client: Synnax,
  { configured = true, properties }: CreateHTTPDeviceOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    name: uniqueName("http_server"),
    make: MAKE,
    model: "HTTP server",
    configured,
    properties: { ...deep.copy(ZERO_PROPERTIES), ...properties },
  });

/**
 * Finds the checkbox input of the pluto switch field labeled with labelText. Pluto
 * switch labels are sibling elements, so role/label queries cannot reach the input.
 */
export const getSwitchInput = (labelText: string): HTMLInputElement =>
  getBySelector<HTMLInputElement>(getInputItem(labelText), "input[type='checkbox']");

/**
 * Finds the icon-only action button inside the pluto header titled title. Task form
 * headers stack multiple icon buttons with no accessible names.
 */
export const getHeaderIconButton = (
  title: string,
  icon: string = "add",
): HTMLButtonElement => {
  const header = screen.getByText(title).closest(".pluto-header");
  if (header == null) throw new Error(`no header titled ${title}`);
  return getIconButton(header, icon);
};
