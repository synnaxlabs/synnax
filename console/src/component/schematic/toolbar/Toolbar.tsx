// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/schematic/toolbar/Toolbar.css";

import { schematic } from "@synnaxlabs/client";
import {
  Breadcrumb,
  Flex,
  Icon,
  Schematic as PSchematic,
  Schematic,
  Tabs,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { EmptyAction, Toolbar as Base } from "@/component";
import { CSS } from "@/component/css";
import { Control } from "@/component/schematic/toolbar/Control";
import { Properties } from "@/component/schematic/toolbar/Properties";
import { Symbols } from "@/component/schematic/toolbar/Symbols";
import { Export } from "@/export";
import { useExport } from "@/service/schematic/export";
import { Session } from "@/session";

const TABS = [
  { tabKey: "symbols", name: "Symbols" },
  { tabKey: "properties", name: "Properties" },
  { tabKey: "control", name: "Control" },
];

const NotEditableContent = (): ReactElement => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const controlState = Session.Schematic.useSelectControlStatus();
  const { canEdit } = Session.Schematic.useSelectEditable();
  const name = Schematic.useSelectName();
  return (
    <EmptyAction
      x
      message={`${name} is not editable.${canEdit ? " To make changes," : ""}`}
      action={
        canEdit
          ? controlState === "acquired"
            ? "release control and enable editing."
            : "enable editing."
          : undefined
      }
      onClick={() => {
        dispatch(Session.Schematic.setEditable({ key, editable: true }));
      }}
    />
  );
};

const Internal = (): ReactElement | null => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const activeTab = Session.Schematic.useSelectActiveToolbarTab();
  const name = Schematic.useSelectName();
  const { isCurrentlyEditable, canEdit } = Session.Schematic.useSelectEditable();
  const handleExport = useExport();
  const selected = Session.Schematic.useSelectSelected();
  const singleSelectedConfig = PSchematic.useSelectElementConfig({
    elKey: selected.length === 1 ? selected[0] : "",
  });
  const singleSelectedName =
    selected.length === 1 &&
    singleSelectedConfig != null &&
    "label" in singleSelectedConfig
      ? (singleSelectedConfig.label?.label ?? null)
      : null;
  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (!isCurrentlyEditable) return <NotEditableContent />;
      switch (tabKey) {
        case "symbols":
          return <Symbols />;
        case "control":
          return <Control />;
        default:
          return <Properties />;
      }
    },
    [isCurrentlyEditable],
  );
  const handleTabSelect = useCallback(
    (tabKey: string): void => {
      dispatch(
        Session.Schematic.selectToolbarTab({
          key,
          tab: tabKey as Session.Schematic.ToolbarTab,
        }),
      );
    },
    [dispatch],
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
      <Base.Content>
        <Base.Header>
          <Breadcrumb.Breadcrumb level="h5">
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Schematic />
              {name}
            </Breadcrumb.Segment>
            {singleSelectedName !== null && (
              <Breadcrumb.Segment weight={400} color={8} level="small">
                {singleSelectedName}
              </Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty className={CSS.BE("schematic", "toolbar", "actions")}>
              <Export.ToolbarButton onExport={() => handleExport(key)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={schematic.ontologyID(key)}
              />
            </Flex.Box>
            {canEdit && (
              <Tabs.Selector
                className={CSS.BE("schematic", "toolbar", "tab-selector")}
              />
            )}
          </Flex.Box>
        </Base.Header>
        <Tabs.Content />
      </Base.Content>
    </Tabs.Provider>
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps) => (
  <Schematic.Suspended schematicKey={layoutKey}>
    <Internal />
  </Schematic.Suspended>
);
