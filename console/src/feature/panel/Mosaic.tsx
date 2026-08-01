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
import {
  Breadcrumb,
  Button,
  Component,
  Dialog,
  Errors,
  Flex,
  Flux,
  Icon,
  Nav,
  Panel,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { TabMenuItems } from "@/feature/panel/ContextMenu";
import { type DeletedFallbackProps, resourceOnly } from "@/feature/panel/fallback";
import { useResetOnRestore } from "@/feature/panel/useResetOnRestore";
import { Empty } from "@/platform";
import { CSS } from "@/platform/css";
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
// the tab.
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
        <Text.Text status="disabled">{description}</Text.Text>
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
      description="This tab references a document that no longer exists."
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
    <Errors.SuspenseBoundary FallbackComponent={TabNameFallback}>
      <Name />
    </Errors.SuspenseBoundary>
  );
};

const Content = (): ReactElement => {
  const tabType = Panel.useSelectTabType({});
  const { Content, Name } = useTab();
  const dispatch = useDispatch();
  const isOverlaid = Session.Panel.useSelectIsTabOverlaid();
  const handleDialogClose = useCallback(
    () => dispatch(Session.Panel.stopOverlaying({})),
    [dispatch],
  );
  return (
    <Errors.SuspenseBoundary FallbackComponent={ContentFallback}>
      <Dialog.Frame
        onVisibleChange={handleDialogClose}
        visible={isOverlaid}
        full
        modalPosition="slammed"
        variant="modal"
        background={isOverlaid ? 0 : undefined}
        className={CSS.BE("panel", "tab-frame")}
      >
        <Dialog.Dialog
          passthrough
          full
          className={CSS(CSS.B(caseconv.toKebab(tabType)), CSS.BE("panel", "tab"))}
        >
          <Nav.Bar
            location="top"
            size="5rem"
            bordered
            className={CSS(
              CSS.B("panel-focus-bar"),
              isOverlaid && CSS.BM("panel-focus-bar", "focused"),
            )}
          >
            {isOverlaid && (
              <>
                <Nav.Bar.Start>
                  <Breadcrumb.Breadcrumb>
                    <Breadcrumb.Segment>
                      <Name />
                    </Breadcrumb.Segment>
                  </Breadcrumb.Breadcrumb>
                </Nav.Bar.Start>
                <Nav.Bar.End pack>
                  <Button.Button onClick={handleDialogClose} size="small" textColor={9}>
                    <Icon.Subtract />
                  </Button.Button>
                </Nav.Bar.End>
              </>
            )}
          </Nav.Bar>
          {/* The dialog force-sizes every direct child to fill it, which would stretch
           * anything the tab renders alongside its main content. The wrapper absorbs
           * that and gives absolutely positioned content the tab as its origin. */}
          <Flex.Box grow empty className={CSS.BE("panel", "tab-content")}>
            <Content />
          </Flex.Box>
        </Dialog.Dialog>
      </Dialog.Frame>
    </Errors.SuspenseBoundary>
  );
};

const content = Component.renderProp(Content);
const tabName = Component.renderProp(TabName);
const extraMenuItems = Component.renderProp(TabMenuItems);

const resolveDroppedTab = (raw: string): panel.NewTab | undefined => {
  const parsed = ontology.idZ.safeParse(raw);
  if (!parsed.success) return undefined;
  return { variant: "resource", resource: parsed.data };
};

const Internal = ({ onCreateTab }: MosaicProps): ReactElement => {
  const selected = Session.Panel.useSelectSelectedTabs();
  const handleSelect = Session.Panel.useSelectTab();
  return (
    <Panel.Mosaic
      className={CSS.B("mosaic")}
      selected={selected}
      onSelect={handleSelect}
      onCreateTab={onCreateTab}
      resolveDroppedTab={resolveDroppedTab}
      extraMenuItems={extraMenuItems}
      rounded={1}
      bordered
      borderColor={5}
      background={0}
      tabName={tabName}
    >
      {content}
    </Panel.Mosaic>
  );
};

const EmptyContent = (): ReactElement => (
  <Flex.Box grow align="center" justify="center">
    <Empty.Action message="No panels open. Create one to get started." />
  </Flex.Box>
);

// Last resort for a panel document that failed to load: the reconcile pass
// should have pruned dead references before the mosaic rendered. Close
// removes the reference the way the prune would have.
const PanelFallback = (props: Errors.FallbackProps): ReactElement => {
  const { error } = props;
  const selected = Session.Panel.useSelectSelected();
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
      {selected != null && (
        <Button.Button onClick={() => dispatch(Session.Panel.remove(selected))}>
          Close
        </Button.Button>
      )}
    </Tombstone>
  );
};

export interface MosaicProps {
  onCreateTab: () => panel.NewTab;
}

export const Mosaic = ({ onCreateTab }: MosaicProps): ReactElement => {
  const selected = Session.Panel.useSelectSelected();
  if (selected == null) return <EmptyContent />;
  return (
    <Errors.SuspenseBoundary key={selected} FallbackComponent={PanelFallback}>
      <Panel.Suspended panelKey={selected}>
        <Internal onCreateTab={onCreateTab} />
      </Panel.Suspended>
    </Errors.SuspenseBoundary>
  );
};
