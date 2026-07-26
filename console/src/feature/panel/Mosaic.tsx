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
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { TabMenuItems } from "@/feature/panel/ContextMenu";
import { Empty } from "@/platform";
import { CSS } from "@/platform/css";
import { Panel as PlatformPanel } from "@/platform/panel";
import { useTab } from "@/platform/panel/tab";
import { Session } from "@/session";

const corpseName = (error: Error): string | undefined =>
  Flux.DeletedError.matches(error)
    ? (error as Flux.DeletedError<{ name?: string }>).corpse.name
    : undefined;

// Tab names render in the selector strip, outside the content's suspense
// boundary. A name service throws when its resource has been deleted, so an
// unguarded name would crash the entire app on a single stale tab.
const TabNameFallback = ({ error }: Errors.FallbackProps): ReactElement => (
  <>
    <Icon.Warning />
    <Text.Text>{corpseName(error) ?? "Not found"}</Text.Text>
  </>
);

// Renders the deleted state of a resource tab: the corpse's name plus Close and,
// for restorable document types, Restore. Remote deletes land here; the tab is
// never closed out from under the user.
const DeletedResourceContent = ({
  error,
  resetErrorBoundary,
}: Errors.FallbackProps): ReactElement => {
  const corpse = (error as Flux.DeletedError<{ name?: string }>).corpse;
  const resource = Panel.useSelectTabResource({});
  const closeTabs = Panel.useCloseResourceTabs();
  const client = Synnax.use();
  const project = Session.Project.useSelectSelected();
  const handleError = Status.useErrorHandler();
  const name = corpse.name ?? "This resource";
  const handleRestore = (): void => {
    handleError(async () => {
      if (client == null) return;
      await PlatformPanel.restore(resource, { client, project, corpse });
      resetErrorBoundary();
    }, `Failed to restore ${name}`);
  };
  return (
    <Flex.Box grow align="center" justify="center" gap="small">
      <Icon.Warning />
      <Text.Text>{name} was deleted</Text.Text>
      <Flex.Box x gap="small">
        <Button.Button onClick={() => closeTabs(resource)}>Close</Button.Button>
        {client != null && PlatformPanel.canRestore(resource) && (
          <Button.Button variant="filled" onClick={handleRestore}>
            Restore
          </Button.Button>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

// A DeletedError can also bubble out of a view tab reading someone else's
// resource; only a resource tab's own deletion gets the tombstone treatment.
const DeletedContent = (props: Errors.FallbackProps): ReactElement => {
  const variant = Panel.useSelectTabVariant({});
  if (variant !== "resource") return <Errors.Fallback {...props} />;
  return <DeletedResourceContent {...props} />;
};

// A reference can permanently outrun its document: the retrieve's not-found
// wait expired without a create broadcast. Offer to close the tab.
const NotFoundResourceContent = (): ReactElement => {
  const resource = Panel.useSelectTabResource({});
  const closeTabs = Panel.useCloseResourceTabs();
  return (
    <Flex.Box grow align="center" justify="center" gap="small">
      <Icon.Warning />
      <Text.Text>This resource could not be found</Text.Text>
      <Button.Button onClick={() => closeTabs(resource)}>Close</Button.Button>
    </Flex.Box>
  );
};

const NotFoundContent = (props: Errors.FallbackProps): ReactElement => {
  const variant = Panel.useSelectTabVariant({});
  if (variant !== "resource") return <Errors.Fallback {...props} />;
  return <NotFoundResourceContent />;
};

// The not-found wait rejects with a wrapper whose cause carries the typed
// error, so the cause is matched alongside the error itself.
const isNotFound = (error: Error): boolean =>
  NotFoundError.matches(error) || NotFoundError.matches(error.cause);

const ContentFallback = (props: Errors.FallbackProps): ReactElement => {
  if (Flux.DeletedError.matches(props.error)) return <DeletedContent {...props} />;
  if (isNotFound(props.error)) return <NotFoundContent {...props} />;
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

export interface MosaicProps {
  onCreateTab: () => panel.NewTab;
}

export const Mosaic = ({ onCreateTab }: MosaicProps): ReactElement => {
  const selected = Session.Panel.useSelectSelected();
  if (selected == null) return <EmptyContent />;
  return (
    <Panel.Suspended panelKey={selected}>
      <Internal onCreateTab={onCreateTab} />
    </Panel.Suspended>
  );
};
