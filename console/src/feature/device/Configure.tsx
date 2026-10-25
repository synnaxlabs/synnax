// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type ComponentType, type ReactElement } from "react";

import { HARDWARE, type Hardware } from "@/feature/device/integrations";
import { Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { type Session } from "@/session";

export interface ConfigureModalParams {
  deviceKey: device.Key;
  /** The key of an entry in {@link HARDWARE}. */
  integration: string;
}

interface HardwareProps {
  deviceKey: device.Key;
  close: () => void;
  /** Returns to the view that opened the form. */
  onBack?: () => void;
  /** Runs after the form opens a new read task for the device. */
  onTaskCreated?: () => void;
}

const createConfigure = ({
  icon,
  zeroProperties,
  useCreateRead,
}: Hardware): ComponentType<HardwareProps> => {
  const ConfigureHardware = ({
    deviceKey,
    close,
    onBack,
    onTaskCreated,
  }: HardwareProps): ReactElement => {
    const createRead = useCreateRead();
    return (
      <PlatformDevice.Configure
        deviceKey={deviceKey}
        close={close}
        onBack={onBack}
        icon={icon}
        initialProperties={zeroProperties}
        onCreateTask={(key) => {
          createRead({ deviceKey: key });
          onTaskCreated?.();
        }}
      />
    );
  };
  return ConfigureHardware;
};

const CONFIGURES: Record<string, ComponentType<HardwareProps> | undefined> =
  Object.fromEntries(HARDWARE.map((entry) => [entry.key, createConfigure(entry)]));

export interface ConfigureHardwareProps extends HardwareProps {
  /** The key of an entry in {@link HARDWARE}. */
  integration: string;
}

/** Configures a device the Driver found, with an action to create a read task next. */
export const ConfigureHardware = ({
  integration,
  ...rest
}: ConfigureHardwareProps): ReactElement => {
  const Entry = CONFIGURES[integration];
  if (Entry == null) throw new Error(`No hardware integration named ${integration}`);
  return <Entry {...rest} />;
};

const Configure = ({
  deviceKey,
  integration,
  close,
}: Session.Modals.ContentProps<ConfigureModalParams, void>): ReactElement => (
  <ConfigureHardware integration={integration} deviceKey={deviceKey} close={close} />
);

/** Opens {@link ConfigureHardware} in a modal. */
export const useConfigureModal = Modals.create<ConfigureModalParams>(Configure);
