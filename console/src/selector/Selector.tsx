// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/selector/Selector.css";

import { type ontology, type panel } from "@synnaxlabs/client";
import { Eraser, Flex, Status, Text } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";

// ResolvedContent is what a selectable hands back when resolving into a panel tab: an
// ontology-backed resource (for visualizations) or an inline view (for arg-driven
// views such as task forms). The panel mosaic routes these to SetTabResource /
// SetTabView respectively.
export type ResolvedContent =
  | { resource: ontology.ID }
  | { view: panel.TabView };

export interface SelectableProps {
  layoutKey: string;
  rename: Modals.PromptRename;
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
  layoutKey: string;
  text: string;
  selectables: Selectable[];
  onResolved?: (content: ResolvedContent) => void;
}

export const Selector = ({
  layoutKey,
  selectables,
  text,
  onResolved,
}: SelectorProps): ReactElement => {
  const place = Layout.usePlacer();
  const rename = Modals.useRename();
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
              layoutKey={layoutKey}
              rename={rename}
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
