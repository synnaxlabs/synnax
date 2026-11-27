// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { suggestIdentifiers } from "@/feature/ni/device/identifier";
import { use, useChangeIdentifier, useSave } from "@/feature/ni/device/queries";
import { INTEGRATION_KEY, MAKE, ZERO_PROPERTIES } from "@/feature/ni/device/types";
import { SCAN_TYPE } from "@/feature/ni/task/types";
import { Device } from "@/platform/device";

const ICON = <Icon.Logo.NI />;

/** Binds the configure form and menu items to the generic device table. */
export const ADAPTER = Device.createLegacyAdapter({
  make: MAKE,
  zeroProperties: ZERO_PROPERTIES,
  suggestIdentifiers,
});

const useConfigurable = (key: ni.device.Key): Device.Configurable => {
  const { name, identifier } = use({ key });
  return {
    key,
    name,
    identifier,
    // A typed device holds no DAQmx alias. The Driver finds it when a task starts.
    location: "",
    configured: identifier !== "",
    suggestIdentifiers: (name) => suggestIdentifiers({ location: "", name }),
  };
};

/** Binds the configure form and menu items to the typed NI device table. */
export const TYPED_ADAPTER: Device.Adapter = {
  make: MAKE,
  resourceID: ni.device.resourceID,
  use: useConfigurable,
  useSave,
  useChangeIdentifier,
};

export const HARDWARE: Device.Hardware = {
  key: INTEGRATION_KEY,
  name: "NI",
  title: "NI hardware",
  description: "DAQ cards, cDAQ chassis, and other NI-DAQmx devices.",
  icon: ICON,
  make: MAKE,
  scanType: SCAN_TYPE,
  location: { shape: "plain" },
  Configure: Device.createConfigure({ icon: ICON, adapter: ADAPTER }),
  checks: [
    "NI-DAQmx is installed on that computer. NI hardware needs Windows or Linux.",
    "The device shows in NI MAX, and no other program reserves it.",
  ],
};

/** The hardware integration whose configure form writes the typed NI device table. */
export const TYPED_HARDWARE: Device.Hardware = {
  ...HARDWARE,
  Configure: Device.createConfigure({ icon: ICON, adapter: TYPED_ADAPTER }),
};
