// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/panel/Mosaic.css";

import { NotFoundError, ontology, type panel } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Button,
  Component,
  Errors,
  Flex,
  Flux,
  Icon,
  Panel,
  Portal,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { memo, type PropsWithChildren, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { TabMenuItems } from "@/feature/panel/ContextMenu";
import { type DeletedFallbackProps, resourceOnly } from "@/feature/panel/fallback";
import { useCreate } from "@/feature/panel/useCreate";
import { useResetOnRestore } from "@/feature/panel/useResetOnRestore";
import { Empty } from "@/platform";
import { CSS } from "@/platform/css";
import { Panel as PlatformPanel } from "@/platform/panel";
import { useTab } from "@/platform/panel/tab";
import { Session } from "@/session";

const corpseName = (error: Error): string | undefined =>
  Flux.DeletedError.matches(error) ? error.corpseName : undefined;

// Tab names render in the selector strip, outside the content's suspense
// boundary. A name service throws when its resource has been deleted, so an
// unguarded name would crash the entire app on a single stale tab.
const TabNameFallbackContent = ({ error }: Errors.FallbackProps): ReactElement => (
  <>
    <Icon.Warning />
    <Text.Text status="warning">{corpseName(error) ?? "Not found"}</Text.Text>
  </>
);

const ResourceTabNameFallback = (props: Errors.FallbackProps): ReactElement => {
  useResetOnRestore(props.resetErrorBoundary);
  return <TabNameFallbackContent {...props} />;
};

const TabNameFallback = resourceOnly(ResourceTabNameFallback, TabNameFallbackContent);

interface TombstoneProps extends PropsWithChildren {
  icon: ReactElement;
  message: string;
  description: string;
}

// Shared shell for terminal tab states (deleted, not found): a dimmed glyph,
// a short heading, one muted line, and an actions row, optically centered in
// the tab the way the mosaic empty state is.
const Tombstone = ({
  icon,
  message,
  description,
  children,
}: TombstoneProps): ReactElement => (
  <Flex.Box center className={CSS.BE("panel", "tombstone")}>
    {/* The centering box fills the tab; this column shrink-wraps so the icon,
     * copy, and actions stay a tight stack instead of spreading across it. */}
    <Flex.Box y align="center" gap={3}>
      <Flex.Box className={CSS.BE("panel", "tombstone-icon")}>{icon}</Flex.Box>
      <Flex.Box y align="center" gap="small">
        <Text.Text level="h5">{message}</Text.Text>
        <Text.Text
          status="disabled"
          className={CSS.BE("panel", "tombstone-description")}
        >
          {description}
        </Text.Text>
      </Flex.Box>
      <Flex.Box x gap="small">
        {children}
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);

// Renders the deleted state of a resource tab: the corpse's name plus Close and,
// for restorable document types, Restore. Every delete lands here, local or
// remote; the tab is never closed out from under the user.
const DeletedResourceContent = ({
  error,
  resetErrorBoundary,
}: DeletedFallbackProps): ReactElement => {
  const resource = Panel.useSelectTabResource({});
  const closeTabs = Panel.useCloseResourceTabs();
  const { restore, Icon: TabIcon } = useTab();
  useResetOnRestore(resetErrorBoundary);
  const client = Synnax.use();
  const project = Session.Project.useSelectSelected();
  const handleError = Status.useErrorHandler();
  const name = error.corpseName ?? "This resource";
  const handleRestore = (): void => {
    handleError(async () => {
      if (client == null || restore == null) return;
      await restore({ client, project, resource });
      resetErrorBoundary();
    }, `Failed to restore ${name}`);
  };
  const restorable = client != null && restore != null;
  return (
    <Tombstone
      icon={<TabIcon />}
      message={`${name} was deleted`}
      description={
        restorable
          ? "Restoring brings it back for everyone."
          : "Close the component to remove it from this panel."
      }
    >
      <Button.Button
        variant="filled"
        status="warning"
        onClick={() => closeTabs(resource)}
      >
        Close
      </Button.Button>
      {restorable && (
        <Button.Button variant="filled" onClick={handleRestore}>
          Restore
        </Button.Button>
      )}
    </Tombstone>
  );
};

const DeletedContent = resourceOnly(DeletedResourceContent);

// A reference can permanently outrun its document: the retrieve's not-found
// wait expired without a create broadcast. Offer to close the tab.
const NotFoundResourceContent = ({
  resetErrorBoundary,
}: Errors.FallbackProps): ReactElement => {
  const resource = Panel.useSelectTabResource({});
  const closeTabs = Panel.useCloseResourceTabs();
  const { Icon: TabIcon } = useTab();
  useResetOnRestore(resetErrorBoundary);
  return (
    <Tombstone
      icon={<TabIcon />}
      message="Resource not found"
      description="This component references a document that no longer exists."
    >
      <Button.Button onClick={() => closeTabs(resource)}>Close</Button.Button>
    </Tombstone>
  );
};

const NotFoundContent = resourceOnly(NotFoundResourceContent);

// The not-found wait rejects with a wrapper whose cause carries the typed
// error, so the cause is matched alongside the error itself.
export const isNotFound = (error: Error): boolean =>
  NotFoundError.matches(error) || NotFoundError.matches(error.cause);

const ContentFallback = (props: Errors.FallbackProps): ReactElement => {
  const { error } = props;
  if (Flux.DeletedError.matches(error))
    return <DeletedContent {...props} error={error} />;
  if (isNotFound(error)) return <NotFoundContent {...props} />;
  return <Errors.Fallback {...props} />;
};

const TabName = (): ReactElement => {
  const { Name } = useTab();
  return (
    <Errors.SuspenseBoundary
      loading={<Icon.Loading />}
      FallbackComponent={TabNameFallback}
    >
      <Name />
    </Errors.SuspenseBoundary>
  );
};

// Both the panel and its tabs are full regions with room for the orbital; the
// tab strip and toolbar keep the inline glyph.
const loading = (
  <Status.Loading>
    <Status.Orbital />
  </Status.Loading>
);

const Content = (): ReactElement => {
  const tabType = Panel.useSelectTabType({});
  const { Content } = useTab();
  return (
    // The box wraps the boundary rather than sitting inside it: the tab's size
    // belongs to the tab, not to whichever of content, loader, or tombstone is
    // currently filling it.
    <Flex.Box
      y
      full
      empty
      className={CSS(CSS.B(caseconv.toKebab(tabType)), CSS.BE("panel", "tab"))}
    >
      <Errors.SuspenseBoundary loading={loading} FallbackComponent={ContentFallback}>
        <Content />
      </Errors.SuspenseBoundary>
    </Flex.Box>
  );
};

const content = Component.renderProp(Content);
const tabName = Component.renderProp(TabName);
const contextMenu = Component.renderProp(TabMenuItems);

const resolveDroppedTab = (raw: string): panel.NewTab | undefined => {
  const parsed = ontology.idZ.safeParse(raw);
  if (!parsed.success) return undefined;
  return { variant: "resource", resource: parsed.data };
};

// Same principle as the no-panel state: the watermark plus a link that opens a
// tab in the scoped panel through the mosaic's regular create flow.
const EmptyTabContent = ({ onCreateTab }: MosaicProps): ReactElement => {
  const openTab = PlatformPanel.useOpenTab();
  const handleCreate = useCallback(
    () => openTab(onCreateTab()),
    [onCreateTab, openTab],
  );
  return (
    <Flex.Box center gap={5} className={CSS.BE("mosaic", "empty-content")}>
      <Logo className="synnax-logo-watermark" />
      <Empty.Action
        x
        className={CSS.BE("mosaic", "empty-action")}
        level="h5"
        message="No components open."
        action="Create a new component"
        onClick={handleCreate}
      />
    </Flex.Box>
  );
};

const Internal = ({ onCreateTab }: MosaicProps): ReactElement => {
  const selected = Session.Panel.useSelectSelectedTabs();
  const handleSelect = Session.Panel.useSelectTab();
  const panelKey = Panel.useOptionalKey();
  const selectedPanel = Session.Panel.useSelectSelected();
  const isOverlaid = Session.Panel.useSelectOverlaid();
  const focusedTab = Session.Panel.useSelectFocusedTab();
  const dispatch = useDispatch();
  // Overlaying only applies to the window's selected panel: kept-alive
  // background panels see the same window-level flag but must not claim tabs.
  const overlaid =
    isOverlaid && panelKey != null && panelKey === selectedPanel
      ? focusedTab
      : undefined;
  const handleStopOverlay = useCallback(
    () => dispatch(Session.Panel.stopOverlaying({})),
    [dispatch],
  );
  return (
    <Panel.Mosaic
      className={CSS.B("mosaic")}
      selected={selected}
      onSelect={handleSelect}
      overlaid={overlaid}
      onStopOverlay={handleStopOverlay}
      onCreateTab={onCreateTab}
      resolveDroppedTab={resolveDroppedTab}
      contextMenu={contextMenu}
      emptyContent={<EmptyTabContent onCreateTab={onCreateTab} />}
      rounded="large"
      bordered
      borderColor={6}
      background={0}
      tabName={tabName}
    >
      {content}
    </Panel.Mosaic>
  );
};

// Mirrors the real mosaic's container chrome so the no-panel state keeps the
// same framed L0 surface instead of collapsing to bare window background.
const EmptyContent = (): ReactElement => {
  const createPanel = useCreate();
  return (
    <Flex.Box
      grow
      align="center"
      justify="center"
      className={CSS(CSS.B("mosaic"), CSS.BM("mosaic", "empty"))}
      rounded="large"
      bordered
      borderColor={6}
      background={0}
    >
      <Flex.Box center gap={5} className={CSS.BE("mosaic", "empty-content")}>
        <Logo className="synnax-logo-watermark" />
        <Empty.Action
          x
          className={CSS.BE("mosaic", "empty-action")}
          level="h5"
          message="No panels open."
          action="Create a new panel"
          onClick={createPanel}
        />
      </Flex.Box>
    </Flex.Box>
  );
};

// Last resort for a panel document that failed to load: the reconcile pass
// should have pruned dead references before the mosaic rendered. Close
// removes the reference the way the prune would have. The panel resolves from
// the surrounding scope: a kept-alive fallback may belong to an unselected panel.
const PanelFallback = (props: Errors.FallbackProps): ReactElement => {
  const { error, resetErrorBoundary } = props;
  const panelKey = Panel.useOptionalKey();
  const invalidate = Panel.useInvalidate();
  const dispatch = useDispatch();
  if (!Flux.DeletedError.matches(error) && !isNotFound(error))
    return <Errors.Fallback {...props} />;
  const name = corpseName(error);
  return (
    <Tombstone
      icon={<Icon.Warning />}
      message={`${name ?? "This panel"} could not be found`}
      description="This window references a panel that no longer exists."
    >
      {panelKey != null && (
        <Button.Button onClick={() => dispatch(Session.Panel.remove(panelKey))}>
          Close
        </Button.Button>
      )}
      {panelKey != null && isNotFound(error) && (
        <Button.Button
          variant="filled"
          // The settled not-found re-throws on every render, so it is discarded
          // before the boundary remounts the panel.
          onClick={() => {
            invalidate({ key: panelKey });
            resetErrorBoundary();
          }}
        >
          Retry
        </Button.Button>
      )}
    </Tombstone>
  );
};

interface KeepAlivePanelProps extends MosaicProps {
  panelKey: panel.Key;
}

// A visited panel renders once into a keyed portal and stays mounted while other
// panels are selected, so switching back reattaches its DOM instead of remounting.
// The scope sits outside the boundary so PanelFallback resolves its own panel.
const KeepAlivePanel = ({
  panelKey,
  onCreateTab,
}: KeepAlivePanelProps): ReactElement => (
  <Portal.In itemKey={panelKey}>
    <Panel.Scope.Provider value={panelKey}>
      <Errors.SuspenseBoundary loading={loading} FallbackComponent={PanelFallback}>
        <Panel.Suspended panelKey={panelKey}>
          <Internal onCreateTab={onCreateTab} />
        </Panel.Suspended>
      </Errors.SuspenseBoundary>
    </Panel.Scope.Provider>
  </Portal.In>
);

const PortaledInPanels = memo(({ onCreateTab }: MosaicProps) => {
  const mounted = Session.Panel.useSelectMounted();
  return (
    <div>
      {mounted.map((key) => (
        <KeepAlivePanel key={key} panelKey={key} onCreateTab={onCreateTab} />
      ))}
    </div>
  );
});
PortaledInPanels.displayName = "PortaledInPanels";

const PortaledOutPanel = memo(() => {
  const selected = Session.Panel.useSelectSelected();
  return selected == null ? (
    <EmptyContent />
  ) : (
    <Portal.Out itemKey={selected} className={CSS.BE("panel", "host")} />
  );
});
PortaledOutPanel.displayName = "PortaledOutPanel";

export interface MosaicProps {
  onCreateTab: () => panel.NewTab;
}

export const Mosaic = ({ onCreateTab }: MosaicProps): ReactElement => (
  <Portal.Context>
    <PortaledInPanels onCreateTab={onCreateTab} />
    <PortaledOutPanel />
  </Portal.Context>
);
