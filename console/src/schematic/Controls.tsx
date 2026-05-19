// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Diagram } from "@synnaxlabs/pluto/diagram";
import { Control } from "@synnaxlabs/pluto/telem/control";
import { location } from "@synnaxlabs/x/location";
import { memo, type ReactElement, useCallback } from "react";

import { Controls as Base } from "@/components";

export interface ControlsProps {
  hasUpdatePermission: boolean;
  controlStatus: Control.Status;
  snapshot: boolean;
}

interface ControlToggleButtonProps {
  control: Control.Status;
}

const ControlToggleButton = ({ control }: ControlToggleButtonProps): ReactElement => {
  const { acquire, release } = Control.useContext();
  const handleChange = useCallback(
    (v: boolean) => (v ? acquire() : release()),
    [acquire, release],
  );
  return (
    <Button.Toggle
      value={control === "acquired"}
      onChange={handleChange}
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      tooltip={`${control === "acquired" ? "Release" : "Acquire"} control`}
    >
      <Icon.Circle />
    </Button.Toggle>
  );
};

export const Controls = memo(
  ({ hasUpdatePermission, controlStatus, snapshot }: ControlsProps): ReactElement => (
    <Base x>
      <Diagram.Controls.SelectViewportMode />
      <Diagram.Controls.FitView />
      <Flex.Box x pack>
        {hasUpdatePermission && (
          <Diagram.Controls.ToggleEdit disabled={controlStatus === "acquired"} />
        )}
        {!snapshot && <ControlToggleButton control={controlStatus} />}
      </Flex.Box>
    </Base>
  ),
);
Controls.displayName = "Controls";
