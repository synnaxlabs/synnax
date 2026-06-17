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
  Tabs,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useTabRenderer } from "@/panel/renderer";
import { useSelectIsOverlaid, useSelectSelectedTabs } from "@/panel/selectors";
import { clearOverlaidTab, focusTab } from "@/panel/slice";

const TabName = (
  props: Omit<Tabs.NameProps, "name" | "onRename" | "tabKey">,
): ReactElement => {
  const nameProps = useTabRenderer().useName();
  const tabKey = Panel.useTabKey("TabName");
  return <Tabs.DefaultName tabKey={tabKey} {...nameProps} {...props} />;
};

const Content = (): ReactElement => {
  const Component = useTabRenderer();
  const dispatch = useDispatch();
  const isOverlaid = useSelectIsOverlaid();
  const handleDialogClose = useCallback(
    () => dispatch(clearOverlaidTab({})),
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
                      {Component.icon}
                      <TabName level="h5" selected={false} editable={false} />
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
  const selected = useSelectSelectedTabs();
  const handleSelect = useCallback(
    (tabKey: string) => dispatch(focusTab({ tabKey })),
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
      contextMenu={}
    >
      {content}
    </Panel.Mosaic>
  );
};
