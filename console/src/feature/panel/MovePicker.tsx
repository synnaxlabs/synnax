// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/panel/MovePicker.css";

import { type panel } from "@synnaxlabs/client";
import { Component, Errors, Icon, List, Panel, Select, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import {
  type TabOrigin,
  useMoveTab,
  useMoveTabToNewPanel,
} from "@/feature/panel/useMoveTab";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

// The row that mints a panel, listed last. Panel keys are UUIDs, so nothing collides.
const NEW_PANEL_KEY = "new";

const ROW_CLASS = CSS.BE("panel-move-picker", "row");

// A panel deleted between the list answer and the retrieve renders no row.
const RowFallback = (): null => null;

const Row = (props: List.ItemProps<panel.Key>): ReactElement => {
  const name = Panel.useName({ key: props.itemKey });
  return (
    <Select.ListItem {...props} align="center" gap="medium" className={ROW_CLASS}>
      <Icon.Panel />
      <Text.Text overflow="ellipsis">{name}</Text.Text>
    </Select.ListItem>
  );
};

const listItem = Component.renderProp(
  (props: List.ItemProps<panel.Key>): ReactElement =>
    props.itemKey === NEW_PANEL_KEY ? (
      <Select.ListItem {...props} align="center" gap="medium" className={ROW_CLASS}>
        <Icon.Add />
        <Text.Text>New panel</Text.Text>
      </Select.ListItem>
    ) : (
      <Errors.SuspenseBoundary loading={null} FallbackComponent={RowFallback}>
        <Row {...props} />
      </Errors.SuspenseBoundary>
    ),
);

export interface MovePickerParams {
  /** The tab being moved, and the panel it currently sits in. */
  origin: TabOrigin;
}

const Content = ({
  origin,
  close,
}: Session.Modals.ContentProps<MovePickerParams, void>): ReactElement => {
  const keys = Session.Panel.useSelectOrderedKeys();
  const moveTab = useMoveTab();
  const moveToNewPanel = useMoveTabToNewPanel();
  const data = useMemo(
    () => [...keys.filter((key) => key !== origin.panel), NEW_PANEL_KEY],
    [keys, origin.panel],
  );
  const handleChange = useCallback(
    (key: panel.Key) => {
      if (key === NEW_PANEL_KEY) moveToNewPanel(origin);
      else moveTab(origin, key);
      close();
    },
    [moveTab, moveToNewPanel, origin, close],
  );
  return (
    <Modals.Frame className={CSS.B("panel-move-picker")}>
      <Modals.Header hideClose icon={<Icon.Panel />}>
        Move to panel
      </Modals.Header>
      <Modals.Body>
        <Select.Frame<panel.Key> data={data} allowNone onChange={handleChange}>
          <List.Items<panel.Key> className={CSS.BE("panel-move-picker", "list")}>
            {listItem}
          </List.Items>
        </Select.Frame>
      </Modals.Body>
    </Modals.Frame>
  );
};

export const useMovePicker = Modals.create<MovePickerParams>(Content);
