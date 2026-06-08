// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel } from "@synnaxlabs/client";
import { Errors, Panel as PlutoPanel, Tabs } from "@synnaxlabs/pluto";
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

const NEW_TAB_NAME = "New Tab";

interface RendererContentProps {
  panelKey: panel.Key;
  tabKey: string;
  type: string;
  layoutKey: string;
  visible: boolean;
}

// RendererContent mounts the layout renderer registered for the content's type,
// keyed by layoutKey — the resource key for a resource tab, the tab key for a view
// tab. Renderers self-load their state from core, so one operator's panel renders
// another operator's content unchanged. Closing the tab removes it from the panel.
const RendererContent = ({
  panelKey,
  tabKey,
  type,
  layoutKey,
  visible,
}: RendererContentProps): ReactElement => {
  const Renderer = Layout.useRenderer(type);
  const { dispatch } = PlutoPanel.useDispatch();
  const handleClose = useCallback(
    () => dispatch({ key: panelKey, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatch, panelKey, tabKey],
  );
  return (
    <Errors.SuspenseBoundary>
      <Renderer
        key={layoutKey}
        layoutKey={layoutKey}
        onClose={handleClose}
        visible={visible}
        focused={false}
      />
    </Errors.SuspenseBoundary>
  );
};

interface SelectorContentProps {
  panelKey: panel.Key;
  tabKey: string;
}

// SelectorContent renders the visualization selector for a tab with no content.
// Picking a type eagerly creates the resource and swaps it onto this tab via
// SetTabResource, keeping the tab's identity and position.
const SelectorContent = ({ panelKey, tabKey }: SelectorContentProps): ReactElement => {
  const { dispatch } = PlutoPanel.useDispatch();
  // A stable key for the resource the selector will create, so re-renders don't
  // churn it. onResolved hands back the created resource's ontology ID.
  const resourceKey = useMemo(() => uuid.create(), [tabKey]);
  const handleResolved = useCallback(
    (resolved: ontology.ID) =>
      dispatch({
        key: panelKey,
        actions: [panel.setTabResource({ key: tabKey, resource: resolved })],
      }),
    [dispatch, panelKey, tabKey],
  );
  return (
    <Selector.Selector
      layoutKey={resourceKey}
      text="Select a Visualization Type"
      selectables={Vis.SELECTABLES}
      onResolved={handleResolved}
    />
  );
};

interface TabContentProps extends PlutoPanel.MosaicTabRenderProps {
  panelKey: panel.Key;
}

// TabContent resolves a panel tab's content union to a renderer: a resource tab
// mounts its type's renderer keyed by the resource key; a view tab mounts its
// type's renderer keyed by the tab key; an empty tab shows the selector.
const TabContent = ({
  panelKey,
  tabKey,
  resource,
  view,
  visible,
}: TabContentProps): ReactElement => {
  if (resource != null)
    return (
      <RendererContent
        panelKey={panelKey}
        tabKey={tabKey}
        type={resource.type}
        layoutKey={resource.key}
        visible={visible}
      />
    );
  if (view != null)
    return (
      <RendererContent
        panelKey={panelKey}
        tabKey={tabKey}
        type={view.type}
        layoutKey={tabKey}
        visible={visible}
      />
    );
  return <SelectorContent panelKey={panelKey} tabKey={tabKey} />;
};

// renderTabName resolves a tab's display name from its content union. A resource
// tab resolves its name from the backing resource (keyed by the resource key); a
// view tab from the view type (keyed by the tab key); an empty tab shows a
// placeholder until the user picks a visualization.
const renderTabName = ({
  resource,
  view,
  ...props
}: PlutoPanel.MosaicTabNameProps): ReactElement => {
  if (resource != null)
    return <Layout.TabName type={resource.type} nameKey={resource.key} {...props} />;
  if (view != null)
    return <Layout.TabName type={view.type} nameKey={props.tabKey} {...props} />;
  return <Tabs.DefaultName {...props} name={NEW_TAB_NAME} />;
};

// Mosaic renders the active panel through pluto's Panel.Mosaic, which owns the tree,
// gestures, and structural dispatch. The console supplies tab content and names
// (resolved from each tab's content union) and the per-window active-tab cursor.
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
      tabName={renderTabName}
      rounded={1}
      bordered
      borderColor={5}
      background={0}
    >
      {(props) => <TabContent panelKey={panelKey} {...props} />}
    </PlutoPanel.Mosaic>
  );
};
