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

export type ResolvedContent = Layout.ResolvedContent;

export interface SelectableProps {
  onPlace: Layout.Placer;
  handleError: Status.ErrorHandler;
  // onResolved, when provided, switches selectables from placing a layout to resolving
  // their content (resource or view) back to the caller. The panel mosaic uses this to
  // fill a null-content tab in place.
  onResolved?: (content: ResolvedContent) => void;
}

export interface Selectable extends FC<SelectableProps> {
  type: string;
  useVisible?: () => boolean;
}

export interface SelectorProps {
  text: string;
  selectables: Selectable[];
  onResolved?: (content: ResolvedContent) => void;
}

export const Selector = ({
  selectables,
  text,
  onResolved,
}: SelectorProps): ReactElement => {
  const place = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return (
    <Eraser.Eraser>
      <Flex.Box
        className={CSS.BE("layout-selector", "frame")}
        gap="large"
        align="center"
        grow
        full
      >
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
              onPlace={place}
              handleError={handleError}
              onResolved={onResolved}
            />
          ))}
        </Flex.Box>
      </Flex.Box>
    </Eraser.Eraser>
  );
};

// createSelector builds a layout renderer for a scoped picker view (e.g. the task-type
// selector). Inside a panel view tab the picker resolves the chosen content into its
// own tab through the RendererView; elsewhere it falls back to placing layouts.
export const createSelector = (
  selectables: Selectable[],
  text: string,
): Layout.Renderer => {
  const C: Layout.Renderer = () => {
    const view = Layout.useRendererView();
    return (
      <Selector selectables={selectables} text={text} onResolved={view?.resolve} />
    );
  };
  C.displayName = "LayoutSelector";
  return C;
};
