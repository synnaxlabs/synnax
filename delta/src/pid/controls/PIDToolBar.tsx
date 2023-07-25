// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { Icon } from "@synnaxlabs/media";
import { Space, Status, Tab, Tabs, Text } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import {
  useSelectPID,
  useSelectPIDEditable,
  useSelectPIDToolbar,
} from "../store/selectors";
import { PIDToolbarTab, setPIDActiveToolbarTab, setPIDEditable } from "../store/slice";

import { PIDElementPropertiesControls } from "./PIDElementPropertiesControls";
import { PIDElements } from "./PIDElementsControls";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { useSelectRequiredLayout } from "@/layout";

export interface PIDToolbarProps {
  layoutKey: string;
}

const TABS = [
  {
    tabKey: "elements",
    name: "Elements",
  },
  {
    tabKey: "properties",
    name: "Properties",
  },
];

interface PIDNotEditableContentProps extends PIDToolbarProps {}

const PIDNotEditableContent = ({
  layoutKey,
}: PIDNotEditableContentProps): ReactElement => {
  const dispatch = useDispatch();
  return (
    <Space.Centered direction="x" size="small">
      <Status.Text variant="disabled" hideIcon>
        PID is not editable. To make changes,
      </Status.Text>
      <Text.Link
        onClick={(e) => {
          e.stopPropagation();
          dispatch(setPIDEditable({ layoutKey, editable: true }));
        }}
        level="p"
      >
        enable edit mode.
      </Text.Link>
    </Space.Centered>
  );
};

export const PIDToolbar = ({ layoutKey }: PIDToolbarProps): ReactElement => {
  const { name } = useSelectRequiredLayout(layoutKey);
  const dispatch = useDispatch();
  const toolbar = useSelectPIDToolbar();
  const editable = useSelectPIDEditable(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tab): ReactElement => {
      if (!editable) return <PIDNotEditableContent layoutKey={layoutKey} />;
      switch (tabKey) {
        case "elements":
          return <PIDElements layoutKey={layoutKey} />;
        default:
          return <PIDElementPropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey, editable]
  );

  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setPIDActiveToolbarTab({ tab: tabKey as PIDToolbarTab }));
    },
    [dispatch]
  );

  return (
    <Space empty>
      <Tabs.Provider
        value={{
          tabs: TABS,
          selected: toolbar.activeTab,
          onSelect: handleTabSelect,
          content,
        }}
      >
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Control />}>{name}</ToolbarTitle>
          <Tabs.Selector style={{ borderBottom: "none" }} size="large" />
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Space>
  );
};
