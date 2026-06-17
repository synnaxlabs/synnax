// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/panel/Mosaic.css";

import {
  Breadcrumb,
  Button,
  Component,
  Dialog,
  Errors,
  Icon,
  Nav,
  Panel,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Session } from "@/panel/session";
import { Tabs } from "@/panel/tabs";

const TabName = (props: Panel.MosaicTabNameRenderProps): ReactElement => {
  const Name = Tabs.useName();
  return <Name {...props} />;
};

const Content = (): ReactElement => {
  const Component = Tabs.useRenderer();
  const dispatch = useDispatch();
  const isOverlaid = Session.useSelectIsOverlaid();
  const handleDialogClose = useCallback(
    () => dispatch(Session.clearOverlaidTab({})),
    [dispatch],
  );
  return (
    <Errors.SuspenseBoundary>
      <Dialog.Frame
        onVisibleChange={handleDialogClose}
        visible={isOverlaid}
        full
        modalPosition="slammed"
        variant="modal"
        background={isOverlaid ? 0 : undefined}
      >
        <Dialog.Dialog passthrough full className={CSS.B("panel-focus")}>
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
                      <Component.Name level="h5" selected={false} />
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
          <Component />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Errors.SuspenseBoundary>
  );
};

const content = Component.renderProp(Content);
const tabName = Component.renderProp(TabName);

export interface MosaicProps {
  panelKey: string;
}

export const Mosaic = ({ panelKey }: MosaicProps): ReactElement => {
  const dispatch = useDispatch();
  const selected = Session.useSelectSelectedTabs();
  const handleSelect = useCallback(
    (tabKey: string) => dispatch(Session.focusTab({ tabKey })),
    [dispatch],
  );
  return (
    <Panel.Mosaic
      panelKey={panelKey}
      selected={selected}
      onSelect={handleSelect}
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
