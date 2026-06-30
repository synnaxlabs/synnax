// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Common } from "@/hardware/common";
import { MAKE } from "@/hardware/ni/device/types";
import { useConfigureModal } from "@/hardware/ni/device/useConfigureModal";

export interface SelectProps extends Omit<
  Common.Device.SelectProps,
  "onConfigure" | "emptyContent" | "make"
> {}

export const Select = (props: SelectProps) => {
  const configure = useConfigureModal();
  return (
    <Common.Device.Select
      {...props}
      onConfigure={(deviceKey) => configure({ deviceKey })}
      emptyContent="No NI devices connected."
      make={MAKE}
      icon={<Icon.Logo.NI />}
    />
  );
};
