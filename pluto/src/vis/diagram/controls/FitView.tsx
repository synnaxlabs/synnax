// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { Icon } from "@/icon";
import { diagram } from "@/vis/diagram/aether";
import { useContext } from "@/vis/diagram/Context";
import { useFitView } from "@/vis/diagram/useFitView";

export interface FitViewProps extends Omit<
  Button.ToggleProps,
  "children" | "onChange" | "value"
> {}

export const FitView = ({ onClick, ...rest }: FitViewProps): ReactElement => {
  const fitView = useFitView();
  const { fitViewOnResize, setFitViewOnResize } = useContext();
  return (
    <Button.Toggle
      onClick={(e) => {
        fitView(diagram.FIT_VIEW_OPTIONS);
        onClick?.(e);
      }}
      tooltip="Fit view to contents"
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
