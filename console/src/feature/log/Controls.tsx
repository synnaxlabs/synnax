// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon, Log as Base, Text, Triggers } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

export const Controls = memo((): ReactElement => {
  const key = Base.useKey();
  const hold = Session.Log.useSelectHold();
  const dispatch = Session.useDispatch();
  const handleHoldChange = (hold: boolean): void => {
    dispatch(Session.Log.setHold({ key, hold }));
  };
  return (
    <Vis.Controls>
      <Button.Toggle
        value={hold}
        onChange={handleHoldChange}
        size="small"
        className={CSS(hold && Vis.CONTROLS_PINNED_CLASS)}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={
          <Text.Text level="small" color={11}>
            {`${hold ? "Resume" : "Pause"} scrolling`}
            <Triggers.Text trigger={Base.PAUSE_TRIGGER} level="small" />
          </Text.Text>
        }
      >
        {hold ? <Icon.Play /> : <Icon.Pause />}
      </Button.Toggle>
    </Vis.Controls>
  );
});
Controls.displayName = "Controls";
