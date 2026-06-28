// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layered/service/arc/editor/toolbar/graph/Toolbar.css";

import { arc } from "@synnaxlabs/client";
import { Access, Arc, Breadcrumb, Flex, Icon, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { Export } from "@/export";
import { Stages } from "@/layered/service/arc/editor/toolbar/graph/Nodes";
import { Properties } from "@/layered/service/arc/editor/toolbar/graph/Properties";
import { useExport } from "@/layered/service/arc/imex/export";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";

const TABS = [
  { tabKey: "stages", name: "Stages" },
  { tabKey: "properties", name: "Properties" },
];

interface NotEditableContentProps {
  name: string;
}

const NotEditableContent = ({ name }: NotEditableContentProps): ReactElement => {
  const key = Arc.useKey();
  const dispatch = useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(key));
  const isEditable = hasUpdatePermission;
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
            dispatch(Session.Arc.setEditable({ key, editable: true }));
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

export const Toolbar = (): ReactElement | null => {
  const key = Arc.useKey();
  const dispatch = useDispatch();
  const { name } = Layout.useSelectRequired(key);
  const toolbar = Session.Arc.useSelectToolbar();
  const editMode = Session.Arc.useSelectEditable();
  const handleExport = useExport();
  const selected = Session.Arc.useSelectSelected();
  const singleNodeKey = selected.length === 1 ? selected[0] : "";
  const singleConfig = Arc.useSelectNodeConfig({ nodeKey: singleNodeKey });
  const selectedName =
    singleConfig != null
      ? (Arc.Graph.Node.REGISTRY[singleConfig.type]?.name ?? null)
      : null;
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(key));
  const canEdit = hasUpdatePermission && editMode;
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!canEdit) return <NotEditableContent name={name} />;
      switch (tabKey) {
        case "stages":
          return <Stages />;
        default:
          return <Properties />;
      }
    },
    [canEdit, name],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(Session.Arc.selectToolbarTab({ key, tab: tabKey as Session.Arc.ToolbarTab }));
    },
    [key, dispatch],
  );
  const contextValue = useMemo(
    () => ({
      tabs: TABS,
      selected: toolbar.selectedTab,
      onSelect: handleTabSelect,
      content,
    }),
    [toolbar.selectedTab, content, handleTabSelect],
  );
  return (
    <Tabs.Provider value={contextValue}>
      <Base.Header>
        <Breadcrumb.Breadcrumb level="h5">
          <Breadcrumb.Segment weight={500} color={10} level="h5">
            <Icon.Arc />
            {name}
          </Breadcrumb.Segment>
          {selectedName != null && (
            <Breadcrumb.Segment weight={400} color={9} level="p">
              {selectedName}
            </Breadcrumb.Segment>
          )}
        </Breadcrumb.Breadcrumb>
        <Flex.Box x align="center" empty>
          <Flex.Box x empty style={{ height: "100%", width: 66 }}>
            <Export.ToolbarButton onExport={() => void handleExport(key)} />
            <Cluster.CopyLinkToolbarButton
              name={name}
              ontologyID={arc.ontologyID(key)}
            />
          </Flex.Box>
          {hasUpdatePermission && (
            <Tabs.Selector style={{ borderBottom: "none", width: 180 }} />
          )}
        </Flex.Box>
      </Base.Header>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
