// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/toolbar/Toolbar.css";

import { schematic } from "@synnaxlabs/client";
import {
  Breadcrumb,
  Flex,
  Icon,
  Panel as PlutoPanel,
  Schematic,
  Tabs,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Control } from "@/feature/schematic/toolbar/Control";
import { Properties } from "@/feature/schematic/toolbar/Properties";
import { Symbols } from "@/feature/schematic/toolbar/Symbols";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";
import { Session } from "@/session";

const NotEditableContent = (): ReactElement => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const controlState = Session.Schematic.useSelectControlStatus();
  const { canEdit } = Session.Schematic.useSelectEditable();
  const name = Schematic.useSelectName();
  return (
    <Empty.Action
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

const Internal = (): ReactElement => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const activeTab = Session.Schematic.useSelectActiveToolbarTab();
  const name = Schematic.useSelectName();
  const { isCurrentlyEditable, canEdit } = Session.Schematic.useSelectEditable();
  const selected = Session.Schematic.useSelectSelected();
  const singleSelectedConfig = Schematic.useSelectElementConfig({
    elKey: selected.length === 1 ? selected[0] : "",
  });
  const singleSelectedName =
    selected.length === 1 &&
    singleSelectedConfig != null &&
    "label" in singleSelectedConfig
      ? (singleSelectedConfig.label?.label ?? null)
      : null;
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
  return (
    <Base.Content>
      <Tabs.Frame value={activeTab} onChange={handleTabSelect} grow>
        <Base.Header>
          <Breadcrumb.Breadcrumb level="h5">
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Schematic />
              {name}
            </Breadcrumb.Segment>
            {singleSelectedName !== null && (
              <Breadcrumb.Segment weight={400} color={9} level="small">
                {singleSelectedName}
              </Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty className={CSS.BE("schematic", "toolbar", "actions")}>
              <Export.ToolbarButton getID={() => schematic.ontologyID(key)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={schematic.ontologyID(key)}
              />
            </Flex.Box>
            {canEdit && (
              <Tabs.Selector className={CSS.BE("schematic", "toolbar", "tab-selector")}>
                <Tabs.Tab itemKey="symbols">Symbols</Tabs.Tab>
                <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
                <Tabs.Tab itemKey="control">Control</Tabs.Tab>
              </Tabs.Selector>
            )}
          </Flex.Box>
        </Base.Header>
        {isCurrentlyEditable ? (
          <>
            <Tabs.Content itemKey="symbols">
              <Symbols />
            </Tabs.Content>
            <Tabs.Content itemKey="properties">
              <Properties />
            </Tabs.Content>
            <Tabs.Content itemKey="control">
              <Control />
            </Tabs.Content>
          </>
        ) : (
          <Tabs.Content>
            <NotEditableContent />
          </Tabs.Content>
        )}
      </Tabs.Frame>
    </Base.Content>
  );
};

export const Toolbar = (): ReactElement => {
  const { key } = PlutoPanel.useSelectTabResource();
  return (
    <Schematic.Suspended schematicKey={key}>
      <Internal />
    </Schematic.Suspended>
  );
};
