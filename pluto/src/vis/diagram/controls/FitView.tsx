// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { useReactFlow } from "@xyflow/react";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { Icon } from "@/icon";
import { Triggers } from "@/triggers";
import { Viewport as BaseViewport } from "@/viewport";
import { diagram } from "@/vis/diagram/aether";
import { useContext } from "@/vis/diagram/Context";

const FIT_VIEW_TRIGGER = BaseViewport.ZOOM_DEFAULT_TRIGGERS.modes.zoomReset[0];

export interface FitViewProps extends Omit<
  Button.ToggleProps,
  "children" | "onChange" | "value"
> {}

export const FitView = ({ onClick, ...rest }: FitViewProps): ReactElement => {
  const { fitView } = useReactFlow();
  const { fitViewOnResize, setFitViewOnResize } = useContext();
  return (
    <Button.Toggle
      onClick={(e) => {
        void fitView(diagram.FIT_VIEW_OPTIONS);
        onClick?.(e);
      }}
      tooltip={tooltip}
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      {...rest}
      value={fitViewOnResize}
      onChange={setFitViewOnResize}
    >
      <Icon.Expand />
    </Button.Toggle>
  );
};

const tooltip = (
  <Triggers.Text trigger={FIT_VIEW_TRIGGER} level="small">
    Fit view to contents
  </Triggers.Text>
);
