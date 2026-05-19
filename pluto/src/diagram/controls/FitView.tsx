// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x/location";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";

import { useReactFlow } from "@xyflow/react";
import { type ReactElement } from "react";

import { diagram } from "@/diagram/aether";
import { useContext } from "@/diagram/Context";

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
      tooltip={<Text.Text level="small">Fit view to contents</Text.Text>}
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
