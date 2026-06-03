// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel } from "@synnaxlabs/client";
import { Flex, Panel as PlutoPanel, Text } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { Selector } from "@/selector";
import { Vis } from "@/vis";

export interface MosaicProps {
  panelKey: panel.Key;
  windowKey: string;
}

interface TabContentProps {
  panelKey: panel.Key;
  tabKey: string;
  resource: ontology.ID | null;
}

// TabContent renders a panel tab. A tab with no resource shows the shared
// visualization selector; picking a type eagerly creates the resource and swaps it
// onto this tab via SetTabResource, keeping the tab's identity and position.
const TabContent = ({ panelKey, tabKey, resource }: TabContentProps): ReactElement => {
  const { dispatch } = PlutoPanel.useDispatch();
  // A stable key for the resource the selector will create, so re-renders don't
  // churn it. onResolved hands back ontologyID(resourceKey).
  const resourceKey = useMemo(() => uuid.create(), [tabKey]);
  const handleResolved = useCallback(
    (resolved: ontology.ID) => {
      dispatch({
        key: panelKey,
        actions: [panel.setTabResource({ key: tabKey, resource: resolved })],
      });
    },
    [dispatch, panelKey, tabKey],
  );
  if (resource == null)
    return (
      <Selector.Selector
        layoutKey={resourceKey}
        text="Select a Visualization Type"
        selectables={Vis.SELECTABLES}
        onResolved={handleResolved}
      />
    );
  // TODO(#9): render the actual visualization from its ontology.ID. Until the
  // resource→viz bridge lands, show what the tab references.
  return (
    <Flex.Box grow center>
      <Text.Text level="p" color={9}>
        {resource.type}:{resource.key.slice(0, 8)}
      </Text.Text>
    </Flex.Box>
  );
};

// Mosaic renders the active panel through pluto's Panel.Mosaic, which owns the tree,
// gestures, and structural dispatch. The console supplies tab content (the selector
// for empty tabs, the visualization otherwise) and the per-window active-tab cursor.
export const Mosaic = ({ panelKey, windowKey }: MosaicProps): ReactElement => {
  const dispatch = useDispatch();
  const activeTab = Layout.useSelectActiveTabKey();
  const handleSelect = useCallback(
    (tabKey: string) => dispatch(Layout.setActiveTab({ windowKey, key: tabKey })),
    [dispatch, windowKey],
  );
  return (
    <PlutoPanel.Mosaic
      panelKey={panelKey}
      activeTab={activeTab ?? undefined}
      onSelect={handleSelect}
      rounded={1}
      bordered
      borderColor={5}
      background={0}
    >
      {({ tabKey, resource }) => (
        <TabContent panelKey={panelKey} tabKey={tabKey} resource={resource} />
      )}
    </PlutoPanel.Mosaic>
  );
};
