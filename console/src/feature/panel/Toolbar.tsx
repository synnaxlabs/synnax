// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors, Flux, Icon, Panel } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { isNotFound } from "@/feature/panel/Mosaic";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { ResourceGuard, useTab } from "@/platform/panel/tab";
import { Toolbar } from "@/platform/toolbar";
import { Session } from "@/session";

interface EmptyContentProps {
  message?: string;
}

const EmptyContent = ({
  message = "No component selected",
}: EmptyContentProps): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header>
      <Toolbar.Title>
        <Icon.Component />
        Component
      </Toolbar.Title>
    </Toolbar.Header>
    <Empty.Action x message={message} />
  </Toolbar.Content>
);

// The toolbar reads the same queries as the tab's content, so a deleted
// resource empties it too. Show a quiet placeholder; the tombstone with Close
// and Restore lives in the mosaic.
const DeletedContent = ({ name }: Flux.Tombstone): ReactElement => (
  <EmptyContent message={`${name ?? "This resource"} was deleted`} />
);

// Deletion is handled by the ResourceGuard, so only the not-found race lands here.
const NotFoundFallback = (props: Errors.FallbackProps): ReactElement => {
  if (!isNotFound(props.error)) return <Errors.Fallback {...props} />;
  return <EmptyContent message="Resource not found" />;
};

// Header for a focused component without its own toolbar: the tab's Name (icon +
// name) with an explanatory empty body. Renames stay with the tab itself.
const NoToolbarContent = (): ReactElement => {
  const { Name } = useTab();
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>
          <Name allowRename={false} />
        </Toolbar.Title>
      </Toolbar.Header>
      <Empty.Action x message="No properties to configure" />
    </Toolbar.Content>
  );
};

const LiveContent = (): ReactElement => {
  const { Toolbar: TabToolbar } = useTab();
  return (
    <Errors.SuspenseBoundary FallbackComponent={NotFoundFallback}>
      {TabToolbar == null ? <NoToolbarContent /> : <TabToolbar />}
    </Errors.SuspenseBoundary>
  );
};

const Content = (): ReactElement => (
  <ResourceGuard FallbackComponent={DeletedContent}>
    <LiveContent />
  </ResourceGuard>
);

const IconFallback = (): ReactElement => <Icon.Component />;

const FocusedTabIcon = (props: Icon.IconProps): ReactElement => {
  const { Icon: TabIcon } = useTab();
  return <TabIcon {...props} />;
};

// The focused tab's Icon, rendered inside its panel and tab scope. Falls back
// to a generic glyph when no tab is focused or the icon fails.
const ToolbarIcon = (props: Icon.IconProps): ReactElement => {
  const panelKey = Session.Panel.useSelectSelected();
  const tabKey = Session.Panel.useSelectFocusedTab(panelKey);
  if (panelKey == null || tabKey == null) return <Icon.Component {...props} />;
  return (
    <Errors.SuspenseBoundary
      key={`${panelKey}:${tabKey}`}
      loading={<Icon.Component {...props} />}
      FallbackComponent={IconFallback}
    >
      <Panel.Suspended panelKey={panelKey}>
        <Panel.TabScope.Provider value={tabKey}>
          <FocusedTabIcon {...props} />
        </Panel.TabScope.Provider>
      </Panel.Suspended>
    </Errors.SuspenseBoundary>
  );
};

// The mosaic shows the tombstone for a missing panel, so the toolbar goes quiet.
const PanelFallback = (props: Errors.FallbackProps): ReactElement => {
  if (!isNotFound(props.error) && !Flux.DeletedError.matches(props.error))
    return <Errors.Fallback {...props} />;
  return <EmptyContent />;
};

const Wrapper = () => {
  const panelKey = Session.Panel.useSelectSelected();
  const tabKey = Session.Panel.useSelectFocusedTab(panelKey);
  if (panelKey == null || tabKey == null) return <EmptyContent />;
  return (
    // Keyed so a latched error boundary never survives a tab switch. Suspended
    // retrieves the panel itself: the toolbar renders outside the mosaic, so it
    // cannot rely on the mosaic having cached the panel first.
    <Errors.SuspenseBoundary
      key={`${panelKey}:${tabKey}`}
      FallbackComponent={PanelFallback}
    >
      <Panel.Suspended panelKey={panelKey}>
        <Panel.TabScope.Provider value={tabKey}>
          <Content />
        </Panel.TabScope.Provider>
      </Panel.Suspended>
    </Errors.SuspenseBoundary>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "tab",
  content: <Wrapper />,
  tooltip: "Component",
  icon: <ToolbarIcon />,
  sizeBounds: { lower: 160, upper: 300 },
  trigger: ["V"],
};
