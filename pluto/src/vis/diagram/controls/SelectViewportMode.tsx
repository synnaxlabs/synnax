// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { Icon } from "@/icon";
import { Select } from "@/select";
import { type Triggers } from "@/triggers";
import { Viewport as BaseViewport } from "@/viewport";
import { useContext } from "@/vis/diagram/Context";

export const VIEWPORT_MODES = ["zoom", "pan", "select"] as const;
const PAN_TRIGGER: Triggers.Trigger[] = [["MouseMiddle"]];
const SELECT_TRIGGER: Triggers.Trigger[] = [["MouseLeft"]];

export const Base = (): ReactElement => {
  const { viewportMode, onViewportModeChange } = useContext();
  return (
    <Select.Buttons
      keys={VIEWPORT_MODES}
      value={viewportMode}
      onChange={onViewportModeChange}
    >
      <Select.Button
        itemKey="pan"
        size="small"
        tooltip={<BaseViewport.TooltipText mode="pan" triggers={PAN_TRIGGER} />}
        tooltipLocation={location.BOTTOM_LEFT}
      >
        <Icon.Pan />
      </Select.Button>
      <Select.Button
        itemKey="select"
        size="small"
        tooltip={<BaseViewport.TooltipText mode="select" triggers={SELECT_TRIGGER} />}
        tooltipLocation={location.BOTTOM_LEFT}
      >
        <Icon.Selection />
      </Select.Button>
    </Select.Buttons>
  );
};

export const SelectViewportMode = memo(Base);
