// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Breadcrumb, Flex, Icon, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { EmptyAction, Toolbar as Core } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/schematic/export";
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
  const hasEditingPermissions = useSelectHasPermission();
  const isSnapshot = useSelectIsSnapshot(layoutKey);
  const isEditable = hasEditingPermissions && !isSnapshot;
  const name = Layout.useSelectRequired(layoutKey).name;
  return (
    <EmptyAction
      x
      message={`${name} is not editable.${isEditable ? " To make changes," : ""}`}
      action={
        controlState === "acquired"
          ? "release control and enable editing."
          : "enable editing."
      }
      onClick={() => {
        dispatch(setEditable({ key: layoutKey, editable: true }));
      }}
    />
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

const tabs = ({ tabKey }: Tabs.Tab): ReactElement => {
  switch (tabKey) {
    case "symbols":
      return <Symbols />;
    case "control":
      return <Control />;
    default:
      return <PropertiesControls />;
  }
};

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const [selectedTab, setSelectedTab] = useState<string>("symbols");
  const value = useMemo(
    () => ({
      tabs: TABS,
      selected: selectedTab,
      onSelect: setSelectedTab,
      content: tabs,
    }),
    [selectedTab],
  );
  return (
    <Tabs.Provider value={value}>
      <Core.Content>
        <Core.Header>
          <Breadcrumb.Breadcrumb level="h5">
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Schematic />
              {name}
            </Breadcrumb.Segment>
            {/* {selectedNames.length === 1 && selectedNames[0] !== null && (
              <Breadcrumb.Segment weight={400} color={8} level="small">
                {selectedNames[0]}
              </Breadcrumb.Segment>
            )} */}
          </Breadcrumb.Breadcrumb>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty style={{ height: "100%", width: 66 }}>
              {/* <Export.ToolbarButton onExport={() => handleExport(layoutKey)} /> */}
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={schematic.ontologyID(layoutKey)}
              />
            </Flex.Box>
            {/* {canEdit && <Tabs.Selector style={{ borderBottom: "none", width: 251 }} />} */}
          </Flex.Box>
        </Core.Header>
        <Tabs.Content />
      </Core.Content>
    </Tabs.Provider>
  );
};
