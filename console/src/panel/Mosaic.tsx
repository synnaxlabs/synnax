// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel } from "@synnaxlabs/client";
import { Errors, Panel as Base, Tabs } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { SELECTABLES } from "@/layouts/Selector";
import { Selector } from "@/selector";

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
  // view is set when the tab's content is an inline view rather than a backing
  // resource. It is exposed to the renderer via RendererView so the renderer can read
  // the view's args/name and write changes back through the panel document.
  view?: panel.TabView;
  visible: boolean;
}

// RendererContent mounts the layout renderer registered for the content's type, keyed
// by layoutKey — the resource key for a resource tab, the tab key for a view tab.
// Resource renderers self-load their state from core; view renderers read their args
// and name from the RendererView (sourced from the tab's view) and write changes back
// through SetTabView. Closing the tab removes it from the panel.
const RendererContent = ({
  panelKey,
  tabKey,
  type,
  layoutKey,
  view,
  visible,
}: RendererContentProps): ReactElement => {
  const Renderer = Layout.useRenderer(type);
  const { dispatch } = Base.useDispatch();
  const { layoutKey: activeTabKey, blurred } = Layout.useSelectActiveMosaicTabState();
  const active = (activeTabKey != null ? tabKey === activeTabKey : visible) && !blurred;
  const handleClose = useCallback(
    () => dispatch({ key: panelKey, actions: [panel.removeTab({ key: tabKey })] }),
    [dispatch, panelKey, tabKey],
  );
  const rendererView = useMemo<Layout.RendererView | null>(
    () =>
      view == null
        ? null
        : {
            key: tabKey,
            name: view.name ?? "",
            args: view.args ?? {},
            update: (patch) =>
              dispatch({
                key: panelKey,
                actions: [
                  panel.setTabView({ key: tabKey, view: { ...view, ...patch } }),
                ],
              }),
          },
    [view, tabKey, panelKey, dispatch],
  );
  const rendered = (
    <Renderer
      key={layoutKey}
      layoutKey={layoutKey}
      onClose={handleClose}
      visible={visible}
      focused={false}
      active={active}
    />
  );
  return (
    <Errors.SuspenseBoundary>
      {rendererView == null ? (
        rendered
      ) : (
        <Layout.RendererViewProvider value={rendererView}>
          {rendered}
        </Layout.RendererViewProvider>
      )}
    </Errors.SuspenseBoundary>
  );
};

interface SelectorContentProps {
  panelKey: panel.Key;
  tabKey: string;
}

// SelectorContent renders the standard component selector for a tab with no content.
// Picking an item resolves it into the tab in place — a resource (SetTabResource) for
// ontology-backed visualizations, or an inline view (SetTabView) for arg-driven views
// such as task forms — keeping the tab's identity and position.
const SelectorContent = ({ panelKey, tabKey }: SelectorContentProps): ReactElement => {
  const { dispatch } = Base.useDispatch();
  // A stable key for the resource the selector may create, so re-renders don't churn
  // it. onResolved hands back the created content (resource or view).
  const resourceKey = useMemo(() => uuid.create(), [tabKey]);
  const handleResolved = useCallback(
    (resolved: Selector.ResolvedContent) =>
      dispatch({
        key: panelKey,
        actions: [
          "resource" in resolved
            ? panel.setTabResource({ key: tabKey, resource: resolved.resource })
            : panel.setTabView({ key: tabKey, view: resolved.view }),
        ],
      }),
    [dispatch, panelKey, tabKey],
  );
  return (
    <Selector.Selector
      layoutKey={resourceKey}
      text="Select a Component Type"
      selectables={SELECTABLES}
      onResolved={handleResolved}
    />
  );
};

interface TabContentProps extends Base.MosaicTabRenderProps {
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
        view={view}
        visible={visible}
      />
    );
  return <SelectorContent panelKey={panelKey} tabKey={tabKey} />;
};

// ViewTabName renders a view tab's name from the view itself (a view has no backing
// resource), renaming through SetTabView. Falls back to a placeholder when the view
// carries no name yet.
const ViewTabName = ({
  panelKey,
  view,
  tabKey,
  name: _name,
  onRename: _onRename,
  ...props
}: Tabs.NameProps & {
  panelKey: panel.Key;
  view: panel.TabView | null;
}): ReactElement => {
  const { dispatch } = Base.useDispatch();
  const handleRename = useCallback(
    (_: string, name: string) => {
      if (view == null) return;
      dispatch({
        key: panelKey,
        actions: [panel.setTabView({ key: tabKey, view: { ...view, name } })],
      });
    },
    [dispatch, panelKey, tabKey, view],
  );
  const name = view?.name ?? "";
  return (
    <Tabs.DefaultName
      tabKey={tabKey}
      name={name === "" ? NEW_TAB_NAME : name}
      onRename={handleRename}
      {...props}
    />
  );
};

// Mosaic renders the active panel through pluto's Panel.Mosaic, which owns the tree,
// gestures, and structural dispatch. The console supplies tab content and names
// (resolved from each tab's content union) and the per-window active-tab cursor.
export const Mosaic = ({ panelKey, windowKey }: MosaicProps): ReactElement => {
  const dispatch = useDispatch();
  const activeTab = Layout.useSelectActiveTabKey();
  const renderers = Layout.useRenderers();
  const unsaved = Layout.useSelectTabUnsavedChanges();
  const handleSelect = useCallback(
    (tabKey: string) => dispatch(Layout.setActiveTab({ windowKey, key: tabKey })),
    [dispatch, windowKey],
  );
  // tabInfo resolves a tab's icon (from the content type's registered renderer) and its
  // unsaved-changes marker (session state), baked onto the tab spec by pluto.
  const tabInfo = useCallback(
    ({
      tabKey,
      resource,
      view,
    }: {
      tabKey: string;
      resource: ontology.ID | null;
      view: panel.TabView | null;
    }) => {
      const type = resource?.type ?? view?.type;
      return {
        icon: type != null ? renderers[type]?.icon : undefined,
        unsavedChanges: unsaved[tabKey] === true,
      };
    },
    [renderers, unsaved],
  );
  // renderTabName resolves a tab's display name from its content union. A resource tab
  // resolves its name through the backing resource's name hook; a view tab from the
  // view's own name; an empty tab shows a placeholder until the user picks a component.
  const renderTabName = useCallback(
    ({ resource, view, ...props }: Base.MosaicTabNameProps): ReactElement => {
      if (resource != null)
        return (
          <Layout.TabName type={resource.type} nameKey={resource.key} {...props} />
        );
      if (view != null)
        return <ViewTabName panelKey={panelKey} view={view} {...props} />;
      return <Tabs.DefaultName {...props} name={NEW_TAB_NAME} />;
    },
    [panelKey],
  );
  return (
    <Base.Mosaic
      panelKey={panelKey}
      activeTab={activeTab ?? undefined}
      onSelect={handleSelect}
      tabName={renderTabName}
      tabInfo={tabInfo}
      rounded={1}
      bordered
      borderColor={5}
      background={0}
    >
      {(props) => <TabContent panelKey={panelKey} {...props} />}
    </Base.Mosaic>
  );
};
