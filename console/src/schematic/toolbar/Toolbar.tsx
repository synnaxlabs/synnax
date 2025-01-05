// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Breadcrumb, Status, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { useExport } from "@/schematic/export";
import {
  useSelectControlStatus,
  useSelectHasPermission,
  useSelectIsSnapshot,
  useSelectOptional,
  useSelectSelectedElementNames,
  useSelectToolbar,
} from "@/schematic/selectors";
import { setActiveToolbarTab, setEditable, type ToolbarTab } from "@/schematic/slice";
import { PropertiesControls } from "@/schematic/toolbar/Properties";
import { Symbols } from "@/schematic/toolbar/Symbols";

const TABS = [
  { tabKey: "symbols", name: "Symbols" },
  { tabKey: "properties", name: "Properties" },
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
    <Align.Center direction="x" size="small">
      <Status.Text variant="disabled" hideIcon>
        {name} is not editable.
        {isEditable ? " To make changes," : ""}
      </Status.Text>
      {isEditable && (
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
      )}
    </Align.Center>
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const toolbar = useSelectToolbar();
  const state = useSelectOptional(layoutKey);
  const handleExport = useExport();
  const selectedNames = useSelectSelectedElementNames(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tabs.Tab): ReactElement => {
      if (!state?.editable) return <NotEditableContent layoutKey={layoutKey} />;
      switch (tabKey) {
        case "symbols":
          return <Symbols layoutKey={layoutKey} />;
        default:
          return <PropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey, state?.editable],
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
      shade: 8,
      level: "h5",
      icon: <Icon.Schematic />,
    },
  ];
  if (selectedNames.length === 1 && selectedNames[0] !== null)
    breadCrumbSegments.push({
      label: selectedNames[0],
      weight: 400,
      shade: 7,
      level: "p",
    });
  if (state == null) return null;
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
        <Align.Space direction="x" empty>
          <Breadcrumb.Breadcrumb level="p">{breadCrumbSegments}</Breadcrumb.Breadcrumb>
        </Align.Space>
        <Align.Space direction="x" align="center" empty>
          <Align.Space direction="x" empty style={{ height: "100%", width: 66 }}>
            <Export.ToolbarButton onExport={() => handleExport(state.key)} />
            <Link.ToolbarCopyButton
              name={name}
              ontologyID={schematic.ontologyID(state.key)}
            />
          </Align.Space>
          {canEdit && <Tabs.Selector style={{ borderBottom: "none", width: 195 }} />}
        </Align.Space>
      </ToolbarHeader>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
