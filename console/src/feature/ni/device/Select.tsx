// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HARDWARE, TYPED_HARDWARE } from "@/feature/ni/device/hardware";
import { TYPED_SOURCE } from "@/feature/ni/device/SelectTyped";
import { MAKE } from "@/feature/ni/device/types";
import { FLAGS } from "@/flags";
import { Device } from "@/platform/device";

export interface SelectProps extends Omit<
  Device.SelectProps,
  "onConfigure" | "emptyContent" | "make" | "connect" | "icon" | "locationFormat"
> {}

const SOURCE = FLAGS.typedNIDevices ? TYPED_SOURCE : undefined;
const CONFIGURE_HARDWARE = FLAGS.typedNIDevices ? TYPED_HARDWARE : HARDWARE;

export const Select = (props: SelectProps) => {
  const configure = Device.useConfigureModal();
  const find = Device.useFindModal();
  // The generic find view returns generic keys, which the typed source cannot read.
  const connect = FLAGS.typedNIDevices
    ? undefined
    : { title: HARDWARE.title, open: () => find({ hardware: HARDWARE }) };
  return (
    <Device.Select
      {...props}
      source={SOURCE}
      onConfigure={(deviceKey) =>
        configure({ deviceKey, hardware: CONFIGURE_HARDWARE })
      }
      emptyContent="No NI devices connected"
      make={MAKE}
      icon={HARDWARE.icon}
      locationFormat={HARDWARE.location}
      connect={connect}
    />
  );
};
