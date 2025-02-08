// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Selector.css";

import { Button, Eraser, type Icon, Status, Text } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type PlacerProps, usePlacer } from "@/layout/hooks";
import { type RendererProps } from "@/layout/slice";
import { type PromptRename, useRename } from "@/modals/Rename";

interface CreateProps {
  layoutKey: string;
  rename: PromptRename;
}

export interface Selectable {
  key: string;
  title: string;
  icon: Icon.Element;
  create: (props: CreateProps) => Promise<PlacerProps | null>;
}

export interface SelectorProps extends Align.SpaceProps, RendererProps {
  layouts?: Selectable[];
  text?: string;
}

const Base = ({
  layoutKey,
  direction,
  layouts,
  visible: _,
  focused: __,
  text = "Select a Component Type",
  ...props
}: SelectorProps): ReactElement => {
  const place = usePlacer();
  const rename = useRename();
  const handleException = Status.useExceptionHandler();
  return (
    <Eraser.Eraser>
      <Align.Center
        className={CSS.B("vis-layout-selector")}
        size="large"
        {...props}
        wrap
      >
        <Text.Text level="h4" shade={6} weight={400}>
          {text}
        </Text.Text>
        <Align.Space
          direction="x"
          wrap
          style={{ maxWidth: "500px", width: "100%" }}
          justify="center"
          size={2.5}
        >
          {layouts?.map(({ key, title, icon, create }) => (
            <Button.Button
              key={key}
              variant="outlined"
              onClick={() => {
                create({ layoutKey, rename })
                  .then((layout) => {
                    if (layout == null) return;
                    place(layout);
                  })
                  .catch(handleException);
              }}
              startIcon={icon}
              style={{ flexBasis: "185px" }}
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
