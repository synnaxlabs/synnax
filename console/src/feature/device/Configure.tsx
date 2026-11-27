// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device } from "@synnaxlabs/client";
import type { ComponentType, ReactElement } from "react";

import { FIND_HARDWARE, HARDWARE, type Hardware } from "@/feature/device/integrations";
import { Modals } from "@/platform/modals";
import type { Session } from "@/session";

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
  /** Runs after the form opens a new read or write task for the device. */
  onTaskCreated?: () => void;
}

const useNoCreate = (): undefined => undefined;

const createConfigure = ({
  Configure,
  useCreateRead,
  useCreateWrite,
}: Hardware): ComponentType<HardwareProps> => {
  const useWrite = useCreateWrite ?? useNoCreate;
  const ConfigureHardware = ({
    deviceKey,
    close,
    onBack,
    onTaskCreated,
  }: HardwareProps): ReactElement => {
    const createRead = useCreateRead();
    const createWrite = useWrite();
    return (
      <Configure
        deviceKey={deviceKey}
        close={close}
        onBack={onBack}
        onCreateReadTask={(key) => {
          createRead({ deviceKey: key });
          onTaskCreated?.();
        }}
        onCreateWriteTask={
          createWrite == null
            ? undefined
            : (key) => {
                createWrite({ deviceKey: key });
                onTaskCreated?.();
              }
        }
      />
    );
  };
  return ConfigureHardware;
};

const CONFIGURES = new Map<Hardware, ComponentType<HardwareProps>>(
  [...HARDWARE, ...FIND_HARDWARE].map((entry) => [entry, createConfigure(entry)]),
);

export interface ConfigureHardwareProps extends HardwareProps {
  /** An entry in {@link HARDWARE} or {@link FIND_HARDWARE}. */
  hardware: Hardware;
}

/** Configures a device the Driver found, with an action to create a read task next. */
export const ConfigureHardware = ({
  hardware,
  ...rest
}: ConfigureHardwareProps): ReactElement => {
  const Entry = CONFIGURES.get(hardware);
  if (Entry == null) throw new Error(`No hardware integration named ${hardware.key}`);
  return <Entry {...rest} />;
};

const Configure = ({
  deviceKey,
  integration,
  close,
}: Session.Modals.ContentProps<ConfigureModalParams, void>): ReactElement => {
  const hardware = HARDWARE.find(({ key }) => key === integration);
  if (hardware == null) throw new Error(`No hardware integration named ${integration}`);
  return <ConfigureHardware hardware={hardware} deviceKey={deviceKey} close={close} />;
};

/** Opens {@link ConfigureHardware} in a modal. */
export const useConfigureModal = Modals.create<ConfigureModalParams>(Configure);
