// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button, Space, SpaceProps, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { LayoutRenderer, useLayoutPlacer } from "@/layout";
import { createLinePlot } from "@/line/store/slice";
import { createPID } from "@/pid/store/slice";

import "@/vis/components/VisLayoutSelector.css";

export interface VisLayoutSelectorProps extends SpaceProps {
  layoutKey?: string;
}

export const VisLayoutSelector = ({
  layoutKey,
  direction,
  ...props
}: VisLayoutSelectorProps): ReactElement => {
  const place = useLayoutPlacer();

  return (
    <Space.Centered
      className={CSS.B("vis-layout-selector")}
      size="large"
      {...props}
      wrap
    >
      <Text level="h4" color="var(--pluto-gray-p0)">
        Select a visualization type
      </Text>
      <Space direction={direction}>
        <Button
          variant="outlined"
          onClick={() => place(createLinePlot({ key: layoutKey }))}
          startIcon={<Icon.Visualize />}
        >
          Line Plot
        </Button>
        <Button
          variant="outlined"
          onClick={() => place(createPID({ key: layoutKey }))}
          startIcon={<Icon.Acquire />}
        >
          PID
        </Button>
      </Space>
    </Space.Centered>
  );
};

export const VisLayoutSelectorRenderer: LayoutRenderer = ({ layoutKey }) => (
  <VisLayoutSelector layoutKey={layoutKey} />
);
