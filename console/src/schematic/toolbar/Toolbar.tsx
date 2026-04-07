// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  Access,
  Breadcrumb,
  Flex,
  Icon,
  Schematic as Base,
  Tabs,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { EmptyAction, Toolbar as CoreToolbar } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/schematic/export";
import {
  useSelectActiveToolbarTab,
  useSelectControlStatus,
  useSelectEditable,
  useSelectSelected,
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
  const hasUpdatePermission = Access.useUpdateGranted(schematic.ontologyID(layoutKey));
  const isSnapshot = Base.useSelectSnapshot({ key: layoutKey }) ?? false;
  const isEditable = hasUpdatePermission && !isSnapshot;
  const name = Layout.useSelectRequired(layoutKey).name;
  return (
    <EmptyAction
      x
      message={`${name} is not editable.${isEditable ? " To make changes," : ""}`}
      action={
        isEditable
          ? controlState === "acquired"
            ? "release control and enable editing."
            : "enable editing."
          : undefined
      }
      onClick={() => dispatch(setEditable({ key: layoutKey, editable: true }))}
    />
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const activeTab = useSelectActiveToolbarTab(layoutKey);
  const editable = useSelectEditable(layoutKey);
  const handleExport = useExport();
  const selected = useSelectSelected(layoutKey);
  const selectedNames = Base.useSelectElementNames({ key: layoutKey, keys: selected });
  const hasUpdatePermission = Access.useUpdateGranted(schematic.ontologyID(layoutKey));
  const isSnapshot = Base.useSelectSnapshot({ key: layoutKey }) ?? false;
  const hasEditPermission = hasUpdatePermission && !isSnapshot;
  const canEdit = hasEditPermission && editable;
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!canEdit) return <NotEditableContent layoutKey={layoutKey} />;
      switch (tabKey) {
        case "symbols":
          return <Symbols layoutKey={layoutKey} />;
        case "control":
          return <Control layoutKey={layoutKey} />;
        default:
          return <PropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey, canEdit],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ key: layoutKey, tab: tabKey as ToolbarTab }));
    },
    [dispatch, layoutKey],
  );
  const value = useMemo(
    () => ({
      tabs: TABS,
      selected: activeTab,
      onSelect: handleTabSelect,
      content,
    }),
    [activeTab, content, handleTabSelect],
  );
  return (
    <Tabs.Provider value={value}>
      <CoreToolbar.Content>
        <CoreToolbar.Header>
          <Breadcrumb.Breadcrumb level="h5">
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Schematic />
              {name}
            </Breadcrumb.Segment>
            {selectedNames.length === 1 && selectedNames[0] !== null && (
              <Breadcrumb.Segment weight={400} color={8} level="small">
                {selectedNames[0]}
              </Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty style={{ height: "100%", width: 66 }}>
              <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={schematic.ontologyID(layoutKey)}
              />
            </Flex.Box>
            {hasEditPermission && (
              <Tabs.Selector style={{ borderBottom: "none", width: 251 }} />
            )}
          </Flex.Box>
        </CoreToolbar.Header>
        <Tabs.Content />
      </CoreToolbar.Content>
    </Tabs.Provider>
  );
};
