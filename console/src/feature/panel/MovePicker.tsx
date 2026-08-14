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
import { Button, Errors, Flex, Icon, Panel, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import {
  type TabOrigin,
  useMoveTab,
  useMoveTabToNewPanel,
} from "@/feature/panel/useMoveTab";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

interface RowProps {
  panelKey: panel.Key;
  onSelect: (key: panel.Key) => void;
}

// A panel deleted between the list answer and the retrieve renders no row.
const RowFallback = (): null => null;

const Row = ({ panelKey, onSelect }: RowProps): ReactElement => (
  <Errors.SuspenseBoundary loading={null} FallbackComponent={RowFallback}>
    <RowContent panelKey={panelKey} onSelect={onSelect} />
  </Errors.SuspenseBoundary>
);

const RowContent = ({ panelKey, onSelect }: RowProps): ReactElement => {
  const name = Panel.useName({ key: panelKey });
  const handleClick = useCallback(() => onSelect(panelKey), [onSelect, panelKey]);
  return (
    <Button.Button
      variant="text"
      align="center"
      justify="start"
      className={CSS.BE("panel-move-picker", "row")}
      onClick={handleClick}
    >
      <Icon.Panel />
      <Text.Text overflow="ellipsis">{name}</Text.Text>
    </Button.Button>
  );
};

export interface MovePickerParams {
  /** The tab being moved, and the panel it currently sits in. */
  origin: TabOrigin;
}

const Content = ({
  origin,
  close,
}: Session.Modals.ContentProps<MovePickerParams, void>): ReactElement => {
  const projectKey = Session.Project.useSelectSelected();
  const keys = Panel.useKeysByProject({ project: projectKey });
  const moveTab = useMoveTab();
  const moveToNewPanel = useMoveTabToNewPanel();
  const handleSelect = useCallback(
    (key: panel.Key) => {
      moveTab(origin, key);
      close();
    },
    [moveTab, origin, close],
  );
  const handleCreate = useCallback(() => {
    moveToNewPanel(origin);
    close();
  }, [moveToNewPanel, origin, close]);
  return (
    <Modals.Frame>
      <Modals.Header icon={<Icon.Panel />}>Move to panel</Modals.Header>
      <Modals.Body>
        <Flex.Box y empty className={CSS.BE("panel-move-picker", "list")}>
          {keys
            .filter((key) => key !== origin.panel)
            .map((key) => (
              <Row key={key} panelKey={key} onSelect={handleSelect} />
            ))}
          <Button.Button
            variant="text"
            align="center"
            justify="start"
            className={CSS.BE("panel-move-picker", "row")}
            onClick={handleCreate}
          >
            <Icon.Add />
            <Text.Text>New panel</Text.Text>
          </Button.Button>
        </Flex.Box>
      </Modals.Body>
    </Modals.Frame>
  );
};

export const useMovePicker = Modals.create<MovePickerParams>(Content);
