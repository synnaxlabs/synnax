// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/LayoutSelector.css";

import { Icon } from "@synnaxlabs/media";
import { Button, Eraser, Text } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { type ReactElement } from "react";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type State } from "@/layout/layout";
import { LinePlot } from "@/lineplot";
import { Schematic } from "@/schematic";

export interface LayoutSelectorProps extends Align.SpaceProps {
  layoutKey?: string;
}

export const LAYOUT_SELECTOR_TYPE = "visLayoutSelector";

export const createLayoutSelector = (
  props: Omit<Partial<State>, "type">,
): Omit<State, "windowKey"> => {
  const {
    location = "mosaic",
    name = "Visualization",
    key = uuidv4(),
    window,
    tab,
  } = props;
  return {
    type: LAYOUT_SELECTOR_TYPE,
    location,
    name,
    key,
    window,
    tab,
  };
};

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
        <Text.Text level="h4" color="var(--pluto-gray-l6)">
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
            onClick={() => place(Schematic.create({ key: layoutKey }))}
            startIcon={<Icon.Schematic />}
          >
            Schematic
          </Button.Button>
        </Align.Space>
      </Align.Center>
    </Eraser.Eraser>
  );
};
