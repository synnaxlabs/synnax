// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Selector.css";

import { Align, Button, Eraser, type Icon, Status, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { type Renderer, type RendererProps } from "@/layout/slice";
import { type PlacerArgs, usePlacer } from "@/layout/usePlacer";
import { Modals } from "@/modals";

export interface SelectableCreateArgs {
  layoutKey: string;
  rename: Modals.PromptRename;
}

export interface Selectable {
  key: string;
  title: string;
  icon: Icon.Element;
  create: (props: SelectableCreateArgs) => Promise<PlacerArgs | null>;
}

export interface SelectorProps extends RendererProps {
  text: string;
  selectables: Selectable[];
}

export const Selector = ({ layoutKey, selectables, text, ...rest }: SelectorProps) => {
  const place = usePlacer();
  const rename = Modals.useRename();
  const handleException = Status.useExceptionHandler();
  return (
    <Eraser.Eraser>
      <Align.Center
        className={CSS.B("vis-layout-selector")}
        size="large"
        {...rest}
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
          {selectables.map(({ key, title, icon, create }) => (
            <Button.Button
              key={key}
              variant="outlined"
              onClick={() => {
                create({ layoutKey, rename })
                  .then((layout) => {
                    if (layout != null) place(layout);
                  })
                  .catch((e) => handleException(e, "Failed to select layout"));
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

export const createSelector = (selectables: Selectable[], text: string): Renderer => {
  const C: Renderer = (props) => (
    <Selector {...props} selectables={selectables} text={text} />
  );
  C.displayName = "LayoutSelector";
  return C;
};
