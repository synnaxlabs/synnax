// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/selector/Selector.css";

import { Eraser, Flex, Text } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import { type Tabs } from "@/panel/tabs/index";

// SELECTOR_VIEW_TYPE is the view type of the component selector. A selector tab is
// just a view tab the host seeds into otherwise-empty leaves; picking an item
// replaces that tab in place.
export const SELECTOR_VIEW_TYPE = "selector";

export interface SelectableProps {
  // tabKey is the panel tab the selectable replaces when picked. Absent outside a
  // panel (e.g. a toolbar create button), in which case the selectable places a
  // layout instead.
  tabKey?: string;
}

export interface Selectable extends FC<SelectableProps> {
  type: string;
  useVisible?: () => boolean;
}

export interface SelectorProps {
  text: string;
  selectables: Selectable[];
  tabKey?: string;
}

export const Selector = ({
  selectables,
  text,
  tabKey,
}: SelectorProps): ReactElement => (
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
          <Selectable key={Selectable.type} tabKey={tabKey} />
        ))}
      </Flex.Box>
    </Flex.Box>
  </Eraser.Eraser>
);

// createSelector builds a view renderer for a scoped picker (e.g. the component
// selector or the task-type selector). It renders the selectables for the tab it
// is mounted in; picking one replaces that tab with the chosen content.
export const createSelector = (
  selectables: Selectable[],
  text: string,
): Tabs.Renderer => {
  const C: Tabs.Renderer = ({ layoutKey }) => (
    <Selector selectables={selectables} text={text} tabKey={layoutKey} />
  );
  C.displayName = "LayoutSelector";
  return C;
};
