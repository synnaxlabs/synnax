// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Selector.css";

import { Button, Eraser, Text } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { PlacerArgs, usePlacer } from "@/layout/hooks";
import { RendererProps } from "@/layout/layout";

export interface Selectable {
  key: string;
  title: string;
  icon: ReactElement;
  create: (layoutKey: string) => PlacerArgs;
}

export interface SelectorProps extends Align.SpaceProps, RendererProps {
  layouts?: Selectable[];
}

const Base = ({
  layoutKey,
  direction,
  layouts,
  ...props
}: SelectorProps): ReactElement => {
  const place = usePlacer();

  return (
    <Eraser.Eraser>
      <Align.Center
        className={CSS.B("vis-layout-selector")}
        size="large"
        {...props}
        wrap
      >
        <Text.Text level="h4" color="var(--pluto-gray-l6)">
          Select a layout type
        </Text.Text>
        <Align.Space direction={direction}>
          {layouts?.map(({ key, title, icon, create }) => (
            <Button.Button
              key={key}
              variant="outlined"
              onClick={() => place(create(layoutKey))}
              startIcon={icon}
            >
              {title}
            </Button.Button>
          ))}
        </Align.Space>
      </Align.Center>
    </Eraser.Eraser>
  );
};

export const createSelectorComponent = (
  layouts: Selectable[],
): ((props: SelectorProps) => ReactElement) => {
  const C = (props: SelectorProps): ReactElement => (
    <Base layouts={layouts} {...props} />
  );
  C.displayName = "LayoutSelector";
  return C;
};
