// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/primitive/arc/editor/toolbar/graph/Toolbar.css";

import { arc } from "@synnaxlabs/client";
import { Arc, Breadcrumb, Flex, Icon, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Stages } from "@/primitive/arc/editor/toolbar/graph/Nodes";
import { Properties } from "@/primitive/arc/editor/toolbar/graph/Properties";
import { Cluster } from "@/primitive/cluster";
import { Export } from "@/primitive/export";
import { Toolbar as Base } from "@/primitive/toolbar";
import { Session } from "@/session";

const TABS = [
  { tabKey: "stages", name: "Stages" },
  { tabKey: "properties", name: "Properties" },
];

const NotEditableContent = (): ReactElement => {
  const key = Arc.useKey();
  const name = Arc.useSelectName();
  const dispatch = Session.useDispatch();
  const { canEdit } = Session.Arc.useSelectEditable();
  return (
    <Flex.Box x gap="small" center>
      <Text.Text status="disabled">
        {name} is not editable.
        {canEdit ? " To make changes," : ""}
      </Text.Text>
      {canEdit && (
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

export interface ToolbarProps {
  onExport: (key: arc.Key) => void;
}

export const Toolbar = ({ onExport }: ToolbarProps): ReactElement | null => {
  const key = Arc.useKey();
  const dispatch = Session.useDispatch();
  const toolbar = Session.Arc.useSelectToolbar();
  const { canEdit, isCurrentlyEditable } = Session.Arc.useSelectEditable();
  const selected = Session.Arc.useSelectSelected();
  const singleNodeKey = selected.length === 1 ? selected[0] : "";
  const singleConfig = Arc.useSelectNodeConfig({ nodeKey: singleNodeKey });
  const name = Arc.useSelectName();
  const handleExport = useCallback(() => onExport(key), [key]);
  const selectedName =
    singleConfig != null
      ? (Arc.Graph.Node.REGISTRY[singleConfig.type]?.name ?? null)
      : null;
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!isCurrentlyEditable) return <NotEditableContent />;
      switch (tabKey) {
        case "stages":
          return <Stages />;
        default:
          return <Properties />;
      }
    },
    [isCurrentlyEditable],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(
        Session.Arc.selectToolbarTab({ key, tab: tabKey as Session.Arc.ToolbarTab }),
      );
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
            <Export.ToolbarButton onExport={handleExport} />
            <Cluster.CopyLinkToolbarButton
              name={name}
              ontologyID={arc.ontologyID(key)}
            />
          </Flex.Box>
          {canEdit && <Tabs.Selector style={{ borderBottom: "none", width: 180 }} />}
        </Flex.Box>
      </Base.Header>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
