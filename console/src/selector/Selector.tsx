// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/selector/Selector.css";

import { Eraser, Flex, Status, Text } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";

export interface SelectableProps {
  layoutKey: string;
  rename: Modals.PromptRename;
  onPlace: Layout.Placer;
  handleError: Status.ErrorHandler;
}

export interface Selectable extends FC<SelectableProps> {
  type: string;
  useVisible?: () => boolean;
}

export interface SelectorProps extends Layout.RendererProps {
  text: string;
  selectables: Selectable[];
}

export const Selector = ({
  layoutKey,
  selectables,
  text,
}: SelectorProps): ReactElement => {
  const place = Layout.usePlacer();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  return (
    <Eraser.Eraser>
      <Flex.Box className={CSS.B("layout-selector")} gap="large" wrap center>
        <Text.Text level="h4" color={10} weight={400}>
          {text}
        </Text.Text>
        <Flex.Box
          x
          wrap
          full="x"
          justify="center"
          gap={2.5}
          className={CSS.BE("layout-selector", "items")}
        >
          {selectables.map((Selectable) => (
            <Selectable
              key={Selectable.type}
              layoutKey={layoutKey}
              rename={rename}
              onPlace={place}
              handleError={handleError}
            />
          ))}
        </Flex.Box>
      </Flex.Box>
    </Eraser.Eraser>
  );
};

export const createSelector = (
  selectables: Selectable[],
  text: string,
): Layout.Renderer => {
  const C: Layout.Renderer = (props) => (
    <Selector {...props} selectables={selectables} text={text} />
  );
  C.displayName = "LayoutSelector";
  return C;
};
