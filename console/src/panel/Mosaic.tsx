// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/panel/Mosaic.css";

import { type panel } from "@synnaxlabs/client";
import {
  Breadcrumb,
  Button,
  Dialog,
  Errors,
  Icon,
  type Menu,
  Nav,
  Panel as Base,
} from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { ContextMenu } from "@/panel/ContextMenu";
import { selectorTab } from "@/panel/selectorTab";
import { TabName } from "@/panel/TabName";

export interface MosaicProps {
  panelKey: panel.Key;
  windowKey: string;
}

interface RendererContentProps {
  panelKey: panel.Key;
  tabKey: string;
}

interface FocusFrameProps extends PropsWithChildren {
  tabKey: string;
  type: string;
}

const FocusFrame = ({ tabKey, type, children }: FocusFrameProps): ReactElement => {
  const dispatch = useDispatch();
  const { windowKey, focused: focusedKey } = Layout.useSelectFocused();
  const renderers = Layout.useRenderers();
  const focused = focusedKey === tabKey;
  const handleFocusChange = useCallback(() => {
    if (windowKey != null) dispatch(Layout.setFocus({ windowKey, key: null }));
  }, [dispatch, windowKey]);
  return (
    <Dialog.Frame
      onVisibleChange={handleFocusChange}
      visible={focused}
      full
      modalPosition="slammed"
      variant="modal"
      background={focused ? 0 : undefined}
    >
      <Dialog.Dialog passthrough full className={CSS.B("panel-focus")}>
        <Nav.Bar
          location="top"
          size="5rem"
          bordered
          className={CSS(
            CSS.B("panel-focus-bar"),
            focused && CSS.BM("panel-focus-bar", "focused"),
          )}
        >
          {focused && (
            <>
              <Nav.Bar.Start>
                <Breadcrumb.Breadcrumb>
                  <Breadcrumb.Segment>
                    {renderers[type]?.icon}
                    <TabName
                      type={type}
                      tabKey={tabKey}
                      level="h5"
                      selected={false}
                      editable={false}
                    />
                  </Breadcrumb.Segment>
                </Breadcrumb.Breadcrumb>
              </Nav.Bar.Start>
              <Nav.Bar.End pack>
                <Button.Button onClick={handleFocusChange} size="small" textColor={9}>
                  <Icon.Subtract />
                </Button.Button>
              </Nav.Bar.End>
            </>
          )}
        </Nav.Bar>
        {children}
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const RendererContent = ({ tabKey, type }: RendererContentProps): ReactElement => {
  const Renderer = Layout.useRenderer(type);
  return (
    <Errors.SuspenseBoundary>
      <FocusFrame tabKey={tabKey} type={type}>
        <Renderer key={tabKey} onClose={handleClose} />
      </FocusFrame>
    </Errors.SuspenseBoundary>
  );
};

export const Mosaic = ({ panelKey, windowKey }: MosaicProps): ReactElement => {
  const dispatch = useDispatch();
  const focused = Layout.useSelectFocusedKey();
  const selected = Layout.useSelectSelectedTabs();
  const handleSelect = useCallback(
    (key: string) => dispatch(Layout.setFocusedTab({ windowKey, key })),
    [dispatch, windowKey],
  );
  const renderContextMenu = useCallback(
    (props: Menu.ContextMenuMenuProps) => (
      <ContextMenu {...props} panelKey={panelKey} />
    ),
    [panelKey],
  );
  const renderTabName = useCallback(
    (props: Base.MosaicTabNameProps): ReactElement => <TabName {...props} />,
    [panelKey],
  );
  return (
    <Base.Mosaic
      panelKey={panelKey}
      focused={focused ?? undefined}
      selected={selected}
      onSelect={handleSelect}
      tabName={renderTabName}
      defaultTab={selectorTab}
      contextMenu={renderContextMenu}
      rounded={1}
      bordered
      borderColor={5}
      background={0}
    >
      {(props) => <TabContent panelKey={panelKey} {...props} />}
    </Base.Mosaic>
  );
};
