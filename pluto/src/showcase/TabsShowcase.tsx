// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DragEventHandler, type ReactElement, useCallback, useState } from "react";

import { Flex } from "@/flex";
import { Haul } from "@/haul";
import { Tabs } from "@/tabs";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

const TAB_TYPE = "showcase-tab";

interface TabItem {
  key: string;
  name: string;
}

const INITIAL_TABS: TabItem[] = [
  { key: "overview", name: "Overview" },
  { key: "channels", name: "Channels" },
  { key: "ranges", name: "Ranges" },
  { key: "logs", name: "Logs" },
  { key: "settings", name: "Settings" },
];

const moveTab = (tabs: TabItem[], key: string, index: number): TabItem[] => {
  const from = tabs.findIndex((t) => t.key === key);
  if (from === -1) return tabs;
  const next = [...tabs];
  const [moved] = next.splice(from, 1);
  next.splice(from < index ? index - 1 : index, 0, moved);
  return next;
};

interface DraggableTabProps {
  tabKey: string;
  children: string;
}

const DraggableTab = ({ tabKey, children }: DraggableTabProps): ReactElement => {
  const { startDrag, onDragEnd } = Haul.useDrag({ type: TAB_TYPE, key: tabKey });
  const handleDragStart = useCallback<DragEventHandler<HTMLDivElement>>(
    (e) => {
      e.dataTransfer.effectAllowed = "move";
      e.dataTransfer.setData("text/plain", tabKey);
      startDrag([{ type: TAB_TYPE, key: tabKey }]);
    },
    [startDrag, tabKey],
  );
  return (
    <Tabs.Tab
      itemKey={tabKey}
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
    >
      {children}
    </Tabs.Tab>
  );
};

interface ReorderableTabsProps {
  variant?: Tabs.Variant;
  vertical?: boolean;
}

const ReorderableTabs = ({
  variant,
  vertical = false,
}: ReorderableTabsProps): ReactElement => {
  const [tabs, setTabs] = useState<TabItem[]>(INITIAL_TABS);
  const [selected, setSelected] = useState<string>(INITIAL_TABS[0].key);
  const handleDrop = useCallback((params: Tabs.SelectorOnDropParams): Haul.Item[] => {
    const items = Haul.filterByType(TAB_TYPE, params.items);
    if (items.length === 0) return [];
    setTabs((prev) => moveTab(prev, String(params.source.key), params.index));
    return items;
  }, []);
  return (
    <Haul.Provider>
      <Tabs.Frame
        value={selected}
        onChange={setSelected}
        x={vertical}
        style={{
          height: vertical ? 200 : 240,
          width: vertical ? 420 : undefined,
          border: "var(--pluto-border)",
          borderRadius: "var(--pluto-border-radius)",
          overflow: "hidden",
        }}
      >
        <Tabs.Selector
          variant={variant}
          haulType={TAB_TYPE}
          onDrop={handleDrop}
          y={vertical}
          style={vertical ? { minWidth: 140 } : undefined}
        >
          {tabs.map((t) => (
            <DraggableTab key={t.key} tabKey={t.key}>
              {t.name}
            </DraggableTab>
          ))}
        </Tabs.Selector>
        {tabs.map((t) => (
          <Tabs.Content key={t.key} itemKey={t.key} center style={{ padding: "3rem" }}>
            <Text.Text level="h4">{t.name}</Text.Text>
          </Tabs.Content>
        ))}
      </Tabs.Frame>
    </Haul.Provider>
  );
};

export const TabsShowcase = (): ReactElement => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Drag to Reorder"
        description="Grab a tab and drag it along the strip. The other tabs slide out of the way to open a gap where it will land, Chrome-style; drop to commit the new order."
      >
        <ReorderableTabs />
      </SubcategorySection>

      <SubcategorySection
        title="Pill Variant"
        description="The same reorder interaction on the pill variant, where tabs are separated rounded buttons."
      >
        <ReorderableTabs variant="pill" />
      </SubcategorySection>

      <SubcategorySection
        title="Vertical Strip"
        description="Reordering along the vertical axis: tabs slide up and down to open the gap."
      >
        <ReorderableTabs vertical />
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
