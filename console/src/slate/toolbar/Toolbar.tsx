// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { slate } from "@synnaxlabs/client";
import { Align, Breadcrumb, Icon, Status, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { Export } from "@/export";
import { useExport } from "@/slate/export";
import {
  useSelectEditable,
  useSelectHasPermission,
  useSelectSelectedElementNames,
  useSelectToolbar,
} from "@/slate/selectors";
import { setActiveToolbarTab, setEditable, type ToolbarTab } from "@/slate/slice";
import { PropertiesControls } from "@/slate/toolbar/Properties";
import { Symbols } from "@/slate/toolbar/Symbols";

const TABS = [
  { tabKey: "symbols", name: "Symbols" },
  { tabKey: "properties", name: "Properties" },
];

interface NotEditableContentProps extends ToolbarProps {}

const NotEditableContent = ({
  layoutKey,
  name,
}: NotEditableContentProps): ReactElement => {
  const dispatch = useDispatch();
  const hasEditingPermissions = useSelectHasPermission();
  const isEditable = hasEditingPermissions;
  return (
    <Align.Center x gap="small">
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
        />
      )}
    </Align.Center>
  );
};

export interface ToolbarProps {
  layoutKey: string;
  name: string;
}

export const Toolbar = ({ layoutKey, name }: ToolbarProps): ReactElement | null => {
  const dispatch = useDispatch();
  const toolbar = useSelectToolbar();
  const editable = useSelectEditable(layoutKey);
  const handleExport = useExport();
  const selectedNames = useSelectSelectedElementNames(layoutKey);
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!editable) return <NotEditableContent layoutKey={layoutKey} name={name} />;
      switch (tabKey) {
        case "symbols":
          return <Symbols layoutKey={layoutKey} />;
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
  const canEdit = useSelectHasPermission();
  const breadCrumbSegments: Breadcrumb.Segments = [
    {
      label: name,
      weight: 500,
      shade: 10,
      level: "h5",
      icon: <Icon.Slate />,
    },
  ];
  if (selectedNames.length === 1 && selectedNames[0] !== null)
    breadCrumbSegments.push({
      label: selectedNames[0],
      weight: 400,
      shade: 9,
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
        <Align.Space x align="center" empty>
          <Align.Space x empty style={{ height: "100%", width: 66 }}>
            <Export.ToolbarButton onExport={() => void handleExport(layoutKey)} />
            <Cluster.CopyLinkToolbarButton
              name={name}
              ontologyID={slate.ontologyID(layoutKey)}
            />
          </Align.Space>
          {canEdit && <Tabs.Selector style={{ borderBottom: "none", width: 180 }} />}
        </Align.Space>
      </Core.Header>
      <Tabs.Content />
    </Tabs.Provider>
  );
};
