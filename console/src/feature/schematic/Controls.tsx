// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Control, Diagram, Flex, Icon, Schematic } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback } from "react";

import { CSS } from "@/platform/css";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

const ControlToggleButton = (): ReactElement => {
  const isAcquired = Session.Schematic.useSelectControlIsAcquired();
  const { acquire, release } = Control.useContext();
  const handleChange = useCallback(
    (v: boolean) => (v ? acquire() : release()),
    [acquire, release],
  );
  return (
    <Button.Toggle
      value={isAcquired}
      onChange={handleChange}
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      tooltip={`${isAcquired ? "Release" : "Acquire"} control`}
    >
      <Icon.Circle />
    </Button.Toggle>
  );
};

export const Controls = memo((): ReactElement => {
  const isSnapshot = Schematic.useSelectSnapshot();
  const isAcquired = Session.Schematic.useSelectControlIsAcquired();
  const { canEdit, isCurrentlyEditable } = Session.Schematic.useSelectEditable();
  return (
    <Vis.Controls x>
      {isCurrentlyEditable && <Diagram.Controls.SelectViewportMode />}
      <Diagram.Controls.FitView />
      <Flex.Box x pack className={CSS(isAcquired && Vis.CONTROLS_PINNED_CLASS)}>
        {canEdit && <Diagram.Controls.ToggleEdit disabled={isAcquired} />}
        {!isSnapshot && <ControlToggleButton />}
      </Flex.Box>
    </Vis.Controls>
  );
});
Controls.displayName = "Schematic.Controls";
