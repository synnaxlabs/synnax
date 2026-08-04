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

import { resourceOnly } from "@/feature/panel/fallback";
import { isNotFound } from "@/feature/panel/Mosaic";
import { useResetOnRestore } from "@/feature/panel/useResetOnRestore";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { NameEditTargetContext, useTab } from "@/platform/panel/tab";
import { Toolbar } from "@/platform/toolbar";
import { Session } from "@/session";

interface EmptyContentProps {
  message?: string;
}

const EmptyContent = ({
  message = "No component selected.",
}: EmptyContentProps): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header>
      <Toolbar.Title icon={<Icon.Component />}>Component</Toolbar.Title>
    </Toolbar.Header>
    <Empty.Action x message={message} />
  </Toolbar.Content>
);

// The toolbar reads the same queries as the tab's content, so a deleted or
// missing resource throws here too. Show a quiet placeholder; the tombstone
// with Close and Restore lives in the mosaic.
const MissingResourceContent = ({
  error,
  resetErrorBoundary,
}: Errors.FallbackProps): ReactElement => {
  useResetOnRestore(resetErrorBoundary);
  const message = Flux.DeletedError.matches(error)
    ? `${error.corpseName ?? "This resource"} was deleted.`
    : "This resource could not be found.";
  return <EmptyContent message={message} />;
};

const MissingContent = resourceOnly(MissingResourceContent);

const Fallback = (props: Errors.FallbackProps): ReactElement => {
  const { error } = props;
  if (!Flux.DeletedError.matches(error) && !isNotFound(error))
    return <Errors.Fallback {...props} />;
  return <MissingContent {...props} />;
};

// Header for a focused component without its own toolbar: the tab's Name (icon +
// name, renames included) with an explanatory empty body.
const NoToolbarContent = (): ReactElement => {
  const { Name } = useTab();
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>
          <NameEditTargetContext value={false}>
            <Name />
          </NameEditTargetContext>
        </Toolbar.Title>
      </Toolbar.Header>
      <Empty.Action x message="This component has no configurable properties." />
    </Toolbar.Content>
  );
};

const Content = (): ReactElement => {
  const { Toolbar: TabToolbar } = useTab();
  return (
    <Errors.SuspenseBoundary FallbackComponent={Fallback}>
      {TabToolbar == null ? <NoToolbarContent /> : <TabToolbar />}
    </Errors.SuspenseBoundary>
  );
};

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
    <Panel.Scope.Provider value={panelKey}>
      <Panel.TabScope.Provider value={tabKey}>
        <Errors.SuspenseBoundary
          key={`${panelKey}:${tabKey}`}
          loading={<Icon.Component {...props} />}
          FallbackComponent={IconFallback}
        >
          <FocusedTabIcon {...props} />
        </Errors.SuspenseBoundary>
      </Panel.TabScope.Provider>
    </Panel.Scope.Provider>
  );
};

const Wrapper = () => {
  const panelKey = Session.Panel.useSelectSelected();
  const tabKey = Session.Panel.useSelectFocusedTab(panelKey);
  if (panelKey == null || tabKey == null) return <EmptyContent />;
  return (
    <Panel.Scope.Provider value={panelKey}>
      <Panel.TabScope.Provider value={tabKey}>
        {/* Keyed so a latched error boundary never survives a tab switch. */}
        <Content key={`${panelKey}:${tabKey}`} />
      </Panel.TabScope.Provider>
    </Panel.Scope.Provider>
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
