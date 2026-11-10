// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { useConfigureModal as useConfigureHardwareModal } from "@/feature/device/Configure";
import { useEditConnectionModal } from "@/feature/device/Connect";
import {
  HARDWARE,
  type Integration,
  INTEGRATIONS,
} from "@/feature/device/integrations";
import { type Tree } from "@/platform/tree";

const BY_MAKE = new Map(INTEGRATIONS.map((i) => [i.make, i]));

/**
 * Returns the integration that connects devices of the given make, or null when no
 * integration in this build does.
 */
export const getIntegration = (make: unknown): Integration | null =>
  typeof make === "string" ? (BY_MAKE.get(make) ?? null) : null;

const DEVICE_ICON = <Icon.Device />;

export const getIcon = (integration: Integration | null): Icon.ReactElement =>
  integration?.icon ?? DEVICE_ICON;

/** Whether the integration's devices are hardware that the Driver finds. */
export const isHardware = ({ key }: Integration): boolean =>
  HARDWARE.some((h) => h.key === key);

/**
 * Returns an opener for the form that sets up a device: the configure form of a
 * hardware integration, or the connect form of a network one.
 */
export const useConfigureModal = (): ((
  integration: Integration,
  deviceKey: device.Key,
) => void) => {
  const configure = useConfigureHardwareModal();
  const connect = useEditConnectionModal();
  return (integration, deviceKey) => {
    if (isHardware(integration)) configure({ deviceKey, integration: integration.key });
    else connect(integration.key, deviceKey);
  };
};

export interface ContextMenuItemsProps extends Tree.ContextMenuProps {
  integration: Integration;
}

/** The context menu items of a device's integration, wired to its configure form. */
export const ContextMenuItems = ({
  integration,
  ...rest
}: ContextMenuItemsProps): ReactElement => {
  const configure = useConfigureModal();
  const { ContextMenuItems: Items } = integration;
  return (
    <Items {...rest} onConfigure={(deviceKey) => configure(integration, deviceKey)} />
  );
};
