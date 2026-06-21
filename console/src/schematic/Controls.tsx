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

import { Controls as Base } from "@/components";
import { useSelectControlIsAcquired, useSelectEditable } from "@/schematic/selectors";

const ControlToggleButton = (): ReactElement => {
  const isAcquired = useSelectControlIsAcquired();
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
  const isSnapshot = Schematic.useSelectSnapshot({});
  const isAcquired = useSelectControlIsAcquired();
  const { canEdit } = useSelectEditable();
  return (
    <Base x>
      <Diagram.Controls.SelectViewportMode />
      <Diagram.Controls.FitView />
      <Flex.Box x pack>
        {canEdit && <Diagram.Controls.ToggleEdit disabled={isAcquired} />}
        {!isSnapshot && <ControlToggleButton />}
      </Flex.Box>
    </Base>
  );
});
Controls.displayName = "Controls";
