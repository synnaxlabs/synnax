// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Breadcrumb, Flex, Icon, Status, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/schematic/export";
import {
  useSelectControlStatus,
  useSelectEditable,
  useSelectHasPermission,
  useSelectIsSnapshot,
  useSelectSelectedElementNames,
  useSelectToolbar,
} from "@/schematic/selectors";
import { setActiveToolbarTab, setEditable, type ToolbarTab } from "@/schematic/slice";
import { Control } from "@/schematic/toolbar/Control";
import { PropertiesControls } from "@/schematic/toolbar/Properties";
import { Symbols } from "@/schematic/toolbar/Symbols";

const TABS = [
  { tabKey: "symbols", name: "Symbols" },
  { tabKey: "properties", name: "Properties" },
  { tabKey: "control", name: "Control" },
];

interface NotEditableContentProps extends ToolbarProps {}

const NotEditableContent = ({ layoutKey }: NotEditableContentProps): ReactElement => {
  const dispatch = useDispatch();
  const controlState = useSelectControlStatus(layoutKey);
  const hasEditingPermissions = useSelectHasPermission();
  const isSnapshot = useSelectIsSnapshot(layoutKey);
  const isEditable = hasEditingPermissions && !isSnapshot;
  const name = Layout.useSelectRequired(layoutKey).name;
  return (
    <Flex.Box x gap="small">
      <Status.Text variant="disabled" hideIcon>
        {name} is not editable.
        {isEditable ? " To make changes," : ""}
      </Status.Text>
      {isEditable && (
        <Text.Link
          onClick={(e: React.MouseEvent<HTMLAnchorElement>) => {
            e.stopPropagation();
            dispatch(setEditable({ key: layoutKey, editable: true }));
          }}
          level="p"
        >
          {controlState === "acquired"
            ? "release control and enable editing."
            : "enable editing."}
        </Text.Link>
      )}
    </Flex.Box>
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const toolbar = useSelectToolbar();
  const isEditable = useSelectEditable(layoutKey) === true;
  const handleExport = useExport();
  const selectedNames = useSelectSelectedElementNames(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!isEditable) return <NotEditableContent layoutKey={layoutKey} />;
      switch (tabKey) {
        case "symbols":
          return <Symbols layoutKey={layoutKey} />;
        case "control":
          return <Control layoutKey={layoutKey} />;
        default:
          return <PropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey, isEditable],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ tab: tabKey as ToolbarTab }));
    },
    [dispatch],
  );
  const canEdit = useSelectHasPermission();
  const breadCrumbSegments: Breadcrumb.Segments = [
    {
      label: name,
      weight: 500,
      shade: 10,
      level: "h5",
      icon: <Icon.Schematic />,
    },
  ];
  if (selectedNames.length === 1 && selectedNames[0] !== null)
    breadCrumbSegments.push({
      label: selectedNames[0],
      weight: 400,
      shade: 8,
      level: "p",
    });
  return (
    <Tabs.Provider
      value={{
        tabs: TABS,
        selected: toolbar.activeTab,
        onSelect: handleTabSelect,
        content,
      }}
    >
      <Core.Header>
        <Breadcrumb.Breadcrumb level="h5">{breadCrumbSegments}</Breadcrumb.Breadcrumb>
        <Flex.Box x align="center" empty>
          <Flex.Box x empty style={{ height: "100%", width: 66 }}>
            <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
            <Cluster.CopyLinkToolbarButton
              name={name}
              ontologyID={schematic.ontologyID(layoutKey)}
            />
          </Flex.Box>
          {canEdit && <Tabs.Selector style={{ borderBottom: "none", width: 251 }} />}
        </Flex.Box>
      </Core.Header>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
