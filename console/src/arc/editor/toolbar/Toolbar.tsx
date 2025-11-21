// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Breadcrumb, Flex, Icon, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { PropertiesControls } from "@/arc/editor/toolbar/Properties";
import { Stages } from "@/arc/editor/toolbar/Stages";
import { useExport } from "@/arc/export";
import {
  useSelectEditable,
  useSelectSelectedElementNames,
  useSelectToolbar,
} from "@/arc/selectors";
import { setActiveToolbarTab, setEditable, type ToolbarTab } from "@/arc/slice";
import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";

const TABS = [
  { tabKey: "stages", name: "Stages" },
  { tabKey: "properties", name: "Properties" },
];

interface NotEditableContentProps extends ToolbarProps {
  name: string;
}

const NotEditableContent = ({
  layoutKey,
  name,
}: NotEditableContentProps): ReactElement => {
  const dispatch = useDispatch();
  const hasEditingPermissions = Access.useGranted({
    objects: arc.ontologyID(layoutKey),
    actions: "create",
  });
  const isEditable = hasEditingPermissions;
  return (
    <Flex.Box x gap="small" center>
      <Text.Text status="disabled">
        {name} is not editable.
        {isEditable ? " To make changes," : ""}
      </Text.Text>
      {isEditable && (
        <Text.Text
          onClick={(e) => {
            e.stopPropagation();
            dispatch(setEditable({ key: layoutKey, editable: true }));
          }}
          variant="link"
          level="p"
          weight={500}
        >
          enable editing.
        </Text.Text>
      )}
    </Flex.Box>
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const dispatch = useDispatch();
  const { name } = Layout.useSelectRequired(layoutKey);
  const toolbar = useSelectToolbar();
  const editable = useSelectEditable(layoutKey);
  const handleExport = useExport();
  const selectedNames = useSelectSelectedElementNames(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!editable) return <NotEditableContent layoutKey={layoutKey} name={name} />;
      switch (tabKey) {
        case "stages":
          return <Stages layoutKey={layoutKey} />;
        default:
          return <PropertiesControls layoutKey={layoutKey} />;
      }
    },
    [layoutKey, editable],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(setActiveToolbarTab({ tab: tabKey as ToolbarTab }));
    },
    [dispatch],
  );
  const canEdit = Access.useGranted({
    objects: arc.ontologyID(layoutKey),
    actions: "create",
  });
  const contextValue = useMemo(
    () => ({
      tabs: TABS,
      selected: toolbar.activeTab,
      onSelect: handleTabSelect,
      content,
    }),
    [toolbar.activeTab, content, handleTabSelect],
  );
  return (
    <Tabs.Provider value={contextValue}>
      <Core.Header>
        <Breadcrumb.Breadcrumb level="h5">
          <Breadcrumb.Segment weight={500} color={10} level="h5">
            <Icon.Arc />
            {name}
          </Breadcrumb.Segment>
          {selectedNames.length === 1 && selectedNames[0] !== null && (
            <Breadcrumb.Segment weight={400} color={9} level="p">
              {selectedNames[0]}
            </Breadcrumb.Segment>
          )}
        </Breadcrumb.Breadcrumb>
        <Flex.Box x align="center" empty>
          <Flex.Box x empty style={{ height: "100%", width: 66 }}>
            <Export.ToolbarButton onExport={() => void handleExport(layoutKey)} />
            <Cluster.CopyLinkToolbarButton
              name={name}
              ontologyID={arc.ontologyID(layoutKey)}
            />
          </Flex.Box>
          {canEdit && <Tabs.Selector style={{ borderBottom: "none", width: 180 }} />}
        </Flex.Box>
      </Core.Header>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
