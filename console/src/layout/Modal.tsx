// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Modals.css";

import { Icon } from "@synnaxlabs/media";
import { Breadcrumb, Button, Modal as Core, Nav } from "@synnaxlabs/pluto";
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
  root?: string;
}

const calculateOffset = (window?: WindowProps): number => {
  if (window?.size?.height == null) return 0;
  if (window?.size?.height < 500) return 15;
  return Math.round(window.size.height / 75);
};

export const Modal = ({ state, remove, root }: ModalProps) => {
  const { key, name, window, icon } = state;
  return (
    <Core.Modal
      key={key}
      visible
      close={() => remove(key)}
      style={layoutCSS(window)}
      root={root}
      offset={calculateOffset(window)}
    >
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
            <Button.Icon onClick={() => remove(key)} size="small">
              <Icon.Close style={{ color: "var(--pluto-gray-l8)" }} />
            </Button.Icon>
          </Nav.Bar.End>
        </Nav.Bar>
      )}
      <Content layoutKey={key} />
    </Core.Modal>
  );
};
