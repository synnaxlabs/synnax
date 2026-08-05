// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors, type Flux, Icon, Panel } from "@synnaxlabs/pluto";
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
  message = "No tab selected.",
}: EmptyContentProps): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header>
      <Toolbar.Title icon={<Icon.Visualize />}>Tab</Toolbar.Title>
    </Toolbar.Header>
    <Empty.Action x message={message} />
  </Toolbar.Content>
);

// The toolbar reads the same queries as the tab's content, so a deleted
// resource empties it too. Show a quiet placeholder; the tombstone with Close
// and Restore lives in the mosaic.
const DeletedContent = ({ name }: Flux.Tombstone): ReactElement => (
  <EmptyContent message={`${name ?? "This resource"} was deleted.`} />
);

// The toolbar reads the same queries as the tab's content, so a missing
// resource throws here too. Deletion is handled by the ResourceGuard.
const NotFoundFallback = (props: Errors.FallbackProps): ReactElement => {
  if (!isNotFound(props.error)) return <Errors.Fallback {...props} />;
  return <EmptyContent message="This resource could not be found." />;
};

const LiveContent = (): ReactElement => {
  const { Toolbar } = useTab();
  if (Toolbar == null) return <EmptyContent />;
  return (
    <Errors.SuspenseBoundary FallbackComponent={NotFoundFallback}>
      <Toolbar />
    </Errors.SuspenseBoundary>
  );
};

const Content = (): ReactElement => (
  <ResourceGuard FallbackComponent={DeletedContent}>
    <LiveContent />
  </ResourceGuard>
);

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
  tooltip: "Tab",
  icon: <Icon.Visualize />,
  sizeBounds: { lower: 160, upper: 300 },
  trigger: ["V"],
};
