// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/toolbar/Toolbar.css";

import { schematic } from "@synnaxlabs/client";
import {
  Breadcrumb,
  Flex,
  Icon,
  Schematic as PSchematic,
  Tabs,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Cluster } from "@/cluster";
import { EmptyAction, Toolbar as Base } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { useExport } from "@/schematic/imex";
import {
  useSelectActiveToolbarTab,
  useSelectControlStatus,
  useSelectEditable,
  useSelectSelected,
  useSelectToolbarTab,
  useSetEditable,
} from "@/schematic/session/slice";
import { Control } from "@/schematic/toolbar/Control";
import { Properties } from "@/schematic/toolbar/Properties";
import { Symbols } from "@/schematic/toolbar/Symbols";
import { Tab } from "@/schematic/tab";

const TABS = [
  { tabKey: "symbols", name: "Symbols" },
  { tabKey: "properties", name: "Properties" },
  { tabKey: "control", name: "Control" },
];

const NotEditableContent = (): ReactElement => {
  const setEditable = useSetEditable();
  const controlState = useSelectControlStatus();
  const name = PSchematic.useSelectName({});
  const [isEditable, canEdit] = useSelectEditable();
  let action: string | undefined;
  if (canEdit)
    if (controlState === "acquired") action = "release control and";
    else action = "enable editing";
  return (
    <EmptyAction
      x
      message={`${name} is not editable.${isEditable ? " To make changes," : ""}`}
      action={action}
      onClick={() => setEditable(true)}
    />
  );
};

export interface ToolbarProps {
  tabKey: string;
}

const useSelectFirstSelectedElementName = () => {
  const selected = useSelectSelected();
  const singleSelectedConfig = PSchematic.useSelectElementConfig({
    elKey: selected.length === 1 ? selected[0] : "",
  });
  return selected.length === 1 &&
    singleSelectedConfig != null &&
    "label" in singleSelectedConfig
    ? (singleSelectedConfig.label?.label ?? null)
    : null;
};

const Internal = (): ReactElement | null => {
  const handleSelectTooblarTab = useSelectToolbarTab();
  const activeTab = useSelectActiveToolbarTab();
  const handleExport = useExport();
  const name = PSchematic.useSelectName({});
  const elementName = useSelectFirstSelectedElementName();
  const key = PSchematic.useKey();
  const [isEditable, canEdit] = useSelectEditable();
  const content = useCallback(({ tabKey }: Tabs.Tab) => {
    if (!isEditable) return <NotEditableContent />;
    switch (tabKey) {
      case "symbols":
        return <Symbols />;
      case "control":
        return <Control />;
      default:
        return <Properties />;
    }
  }, []);
  const value = useMemo(
    () =>
      ({
        tabs: TABS,
        selected: activeTab,
        onSelect: handleSelectTooblarTab,
        content,
      }) as Tabs.ContextValue,
    [activeTab, content, handleSelectTooblarTab],
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
            {elementName !== null && (
              <Breadcrumb.Segment weight={400} color={8} level="small">
                {elementName}
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

export const Toolbar = () => {
  const { key } = Tab.useArgs();
  return (
    <PSchematic.Suspended schematicKey={key}>
      <Internal />;
    </PSchematic.Suspended>
  );
};
