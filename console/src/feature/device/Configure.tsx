// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { HARDWARE } from "@/feature/device/integrations";
import { Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { type Session } from "@/session";

export interface ConfigureModalParams {
  deviceKey: device.Key;
  /** The key of an entry in {@link HARDWARE}. */
  integration: string;
}

const Configure = ({
  deviceKey,
  integration,
  close,
}: Session.Modals.ContentProps<ConfigureModalParams, void>): ReactElement => {
  const entry = HARDWARE.find(({ key }) => key === integration);
  if (entry == null) throw new Error(`No hardware integration named ${integration}`);
  const createRead = entry.useCreateRead();
  return (
    <PlatformDevice.Configure
      deviceKey={deviceKey}
      close={close}
      icon={entry.icon}
      initialProperties={entry.zeroProperties}
      onCreateTask={(key) => createRead({ deviceKey: key })}
    />
  );
};

/** Configures a device the Driver found, with an action to create a read task next. */
export const useConfigureModal = Modals.create<ConfigureModalParams>(Configure);
