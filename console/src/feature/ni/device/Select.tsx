// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { HARDWARE } from "@/feature/ni/device/hardware";
import { MAKE } from "@/feature/ni/device/types";
import { Device } from "@/platform/device";

export interface SelectProps extends Omit<
  Device.SelectProps,
  "onConfigure" | "emptyContent" | "make" | "connect"
> {}

export const Select = (props: SelectProps) => {
  const configure = Device.useConfigureModal();
  const find = Device.useFindModal();
  return (
    <Device.Select
      {...props}
      onConfigure={(deviceKey) => configure({ deviceKey, hardware: HARDWARE })}
      emptyContent="No NI devices connected"
      make={MAKE}
      icon={<Icon.Logo.NI />}
      connect={{ title: HARDWARE.title, open: () => find({ hardware: HARDWARE }) }}
    />
  );
};
