import { Button, Control, Diagram, Flex, Icon } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
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
