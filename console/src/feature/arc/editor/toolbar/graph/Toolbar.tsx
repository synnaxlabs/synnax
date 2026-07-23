// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/toolbar/graph/Toolbar.css";

import { arc } from "@synnaxlabs/client";
import { Arc, Breadcrumb, Flex, Icon, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Stages } from "@/feature/arc/editor/toolbar/graph/Nodes";
import { Properties } from "@/feature/arc/editor/toolbar/graph/Properties";
import { useExport } from "@/feature/arc/export";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";
import { Session } from "@/session";

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

export const Toolbar = (): ReactElement | null => {
  const key = Arc.useKey();
  const dispatch = Session.useDispatch();
  const toolbar = Session.Arc.useSelectToolbar();
  const { canEdit, isCurrentlyEditable } = Session.Arc.useSelectEditable();
  const selected = Session.Arc.useSelectSelected();
  const singleNodeKey = selected.length === 1 ? selected[0] : "";
  const singleConfig = Arc.useSelectNodeConfig({ nodeKey: singleNodeKey });
  const name = Arc.useSelectName();
  const handleExport = useExport();
  const selectedName =
    singleConfig != null
      ? (Arc.Graph.Node.REGISTRY[singleConfig.type]?.name ?? null)
      : null;
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(
        Session.Arc.selectToolbarTab({ key, tab: tabKey as Session.Arc.ToolbarTab }),
      );
    },
    [key, dispatch],
  );
  return (
    <Tabs.Frame value={toolbar.selectedTab} onChange={handleTabSelect} grow>
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
          <Flex.Box x empty className={CSS.BE("arc-toolbar", "actions")}>
            <Export.ToolbarButton onExport={() => handleExport(key)} />
            <Cluster.CopyLinkToolbarButton
              name={name}
              ontologyID={arc.ontologyID(key)}
            />
          </Flex.Box>
          {canEdit && (
            <Tabs.Selector className={CSS.BE("arc-toolbar", "tabs")}>
              <Tabs.Tab itemKey="stages">Stages</Tabs.Tab>
              <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
            </Tabs.Selector>
          )}
        </Flex.Box>
      </Base.Header>
      {isCurrentlyEditable ? (
        <>
          <Tabs.Content itemKey="stages">
            <Stages />
          </Tabs.Content>
          <Tabs.Content itemKey="properties">
            <Properties />
          </Tabs.Content>
        </>
      ) : (
        <Tabs.Content>
          <NotEditableContent />
        </Tabs.Content>
      )}
    </Tabs.Frame>
  );
};
