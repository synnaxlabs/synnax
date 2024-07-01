// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Status, Tabs } from "@synnaxlabs/pluto";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import {
  useSelect,
  useSelectControlStatus,
  useSelectToolbar,
} from "@/schematic/selectors";
import { setActiveToolbarTab, setEditable, type ToolbarTab } from "@/schematic/slice";
import { PropertiesControls } from "@/schematic/toolbar/Properties";
import { Symbols } from "@/schematic/toolbar/Symbols";

export interface ToolbarProps {
  layoutKey: string;
}

const TABS = [
  {
    tabKey: "symbols",
    name: "Symbols",
  },
  {
    tabKey: "properties",
    name: "Properties",
  },
];

interface NotEditableContentProps extends ToolbarProps {}

const NotEditableContent = ({ layoutKey }: NotEditableContentProps): ReactElement => {
  const dispatch = useDispatch();
  const controlState = useSelectControlStatus(layoutKey);
  return (
    <Align.Center direction="x" size="small">
      <Status.Text variant="disabled" hideIcon>
        Schematic is not editable. To make changes,
      </Status.Text>
      <Text.Link
        onClick={(e) => {
          e.stopPropagation();
          dispatch(setEditable({ key: layoutKey, editable: true }));
        }}
        level="p"
      >
        {controlState === "acquired"
          ? "release control and enable edit mode."
          : "enable edit mode."}
      </Text.Link>
    </Align.Center>
  );
};

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const toolbar = useSelectToolbar();
  const schematic = useSelect(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tabs.Tab): ReactElement => {
      if (!schematic.editable) return <NotEditableContent layoutKey={layoutKey} />;
      switch (tabKey) {
        case "symbols":
          return <Symbols layoutKey={layoutKey} />;
        default:
          return <PropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey, schematic?.editable],
  );

  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ tab: tabKey as ToolbarTab }));
    },
    [dispatch],
  );

  if (schematic == null) return null;

  return (
    <Tabs.Provider
      value={{
        tabs: TABS,
        selected: toolbar.activeTab,
        onSelect: handleTabSelect,
        content,
      }}
    >
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Schematic />}>{name}</ToolbarTitle>
        <Tabs.Selector style={{ borderBottom: "none" }} />
      </ToolbarHeader>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
