// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Modals.css";

import { Breadcrumb, Button, Dialog, Icon, Nav } from "@synnaxlabs/pluto";
import { type CSSProperties } from "react";

import { Content } from "@/layout/Content";
import { type State, type WindowProps } from "@/layout/slice";

const layoutCSS = (window?: WindowProps): CSSProperties => ({
  width: "100%",
  height: "100%",
  maxWidth: window?.size?.width,
  maxHeight: window?.size?.height,
  minWidth: window?.minSize?.width,
  minHeight: window?.minSize?.height,
});

interface ModalProps {
  state: State;
  remove: (key: string) => void;
}

const calculateOffset = (window?: WindowProps): number => {
  if (window?.size?.height == null) return 0;
  if (window?.size?.height < 500) return 15;
  return Math.round(window.size.height / 75);
};

export const Modal = ({ state, remove }: ModalProps) => {
  const { key, name, window, icon } = state;
  return (
    <Dialog.Frame
      key={key}
      variant="modal"
      visible
      onVisibleChange={() => remove(key)}
      modalOffset={calculateOffset(window)}
      background={0}
    >
      <Dialog.Dialog style={layoutCSS(window)}>
        {window?.navTop && (
          <Nav.Bar location="top" size="6rem" bordered>
            {(window?.showTitle ?? true) && (
              <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
                <Breadcrumb.Breadcrumb icon={icon} hideFirst={false}>
                  {name}
                </Breadcrumb.Breadcrumb>
              </Nav.Bar.Start>
            )}
            <Nav.Bar.End style={{ paddingRight: "1rem" }}>
              <Button.Button onClick={() => remove(key)} size="small">
                <Icon.Close style={{ color: "var(--pluto-gray-l10)" }} />
              </Button.Button>
            </Nav.Bar.End>
          </Nav.Bar>
        )}
        <Content layoutKey={key} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
