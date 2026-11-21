// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { location } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Viewport as BaseViewport } from "@/viewport";
import { useContext } from "@/vis/diagram/Context";

export const SelectViewportMode = (): ReactElement => {
  const { viewportMode, onViewportModeChange, triggers } = useContext();
  return (
    <Select.Buttons
      variant="outlined"
      value={viewportMode}
      onChange={onViewportModeChange}
    >
      <Select.Item
        itemKey="pan"
        size="small"
        tooltip={<BaseViewport.TooltipText mode="pan" triggers={triggers.modes.pan} />}
        tooltipLocation={location.BOTTOM_LEFT}
      >
        <Icon.Pan />
      </Select.Item>
      <Select.Item
        itemKey="select"
        size="small"
        tooltip={
          <BaseViewport.TooltipText mode="select" triggers={triggers.modes.select} />
        }
        tooltipLocation={location.BOTTOM_LEFT}
      >
        <Icon.Selection />
      </Select.Item>
    </Select.Buttons>
  );
};
