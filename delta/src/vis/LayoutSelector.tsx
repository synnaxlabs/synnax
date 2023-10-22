// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Eraser, Button, Align, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { PID } from "@/pid";

import "@/vis/LayoutSelector.css";

export interface LayoutSelectorProps extends Align.SpaceProps {
  layoutKey?: string;
}

export const LayoutSelector = ({
  layoutKey,
  direction,
  ...props
}: LayoutSelectorProps): ReactElement => {
  const place = Layout.usePlacer();

  return (
    <Eraser.Eraser>
      <Align.Center
        className={CSS.B("vis-layout-selector")}
        size="large"
        {...props}
        wrap
      >
        <Text.Text level="h4" color="var(--pluto-gray-p0)">
          Select a visualization type
        </Text.Text>
        <Align.Space direction={direction}>
          <Button.Button
            variant="outlined"
            onClick={() => place(LinePlot.create({ key: layoutKey }))}
            startIcon={<Icon.Visualize />}
          >
            Line Plot
          </Button.Button>
          <Button.Button
            variant="outlined"
            onClick={() => place(PID.create({ key: layoutKey }))}
            startIcon={<Icon.PID />}
          >
            PID
          </Button.Button>
        </Align.Space>
      </Align.Center>
    </Eraser.Eraser>
  );
};
