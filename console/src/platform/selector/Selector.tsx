// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/selector/Selector.css";

import { type panel } from "@synnaxlabs/client";
import { Eraser, Flex, type Icon, Panel as PPanel, Text } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Panel } from "@/platform/panel";

/** OnSelectParams is passed to a selectable's onSelect hook when it is chosen. */
export interface OnSelectParams {
  /**
   * tabKey is the selector tab the pick was made from, so the selectable can open
   * its content into that tab in place rather than a new one.
   */
  tabKey?: panel.TabKey;
}

export interface SelectableProps {
  tabKey?: panel.TabKey;
}

export interface Selectable extends FC<SelectableProps> {
  type: string;
  useVisible?: () => boolean;
}

export interface CreateParams {
  selectables: Selectable[];
  tabTitle: string;
  text: string;
  icon: Icon.ReactElement;
}

export const create = ({
  selectables,
  tabTitle,
  text,
  icon,
}: CreateParams): Panel.Tab => {
  const Icon: Panel.TabIcon = () => icon;
  const Content: Panel.Content = (): ReactElement => {
    const tabKey = PPanel.useOptionalTabKey();
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
            <Icon />
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
  };
  Content.displayName = `${tabTitle}.Selector`;
  const Name = Panel.createStaticTabName({ name: tabTitle, icon });
  return { Content, Name, Icon };
};
