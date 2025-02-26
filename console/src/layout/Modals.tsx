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
import { Breadcrumb, Button, Menu, Modal as Core, Nav } from "@synnaxlabs/pluto";
import { type CSSProperties, type ReactElement } from "react";

import { Beta } from "@/components";
import { Content } from "@/layout/Content";
import { useSelectModals } from "@/layout/selectors";
import { type State, type WindowProps } from "@/layout/slice";
import { useRemover } from "@/layout/useRemover";
import { DefaultContextMenu } from "@/layout/Window";

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
  centered?: boolean;
  root?: string;
}

const Modal = ({ state, remove, centered, root }: ModalProps): ReactElement => {
  const { key, name, window, icon } = state;
  const menuProps = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
      <Core.Modal
        key={key}
        centered={centered}
        visible
        close={() => remove(key)}
        style={layoutCSS(window)}
        root={root}
      >
        {window?.navTop && (
          <Nav.Bar location="top" size="6rem">
            {(window?.showTitle ?? true) && (
              <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
                <Breadcrumb.Breadcrumb icon={icon} hideFirst={false}>
                  {name}
                </Breadcrumb.Breadcrumb>
              </Nav.Bar.Start>
            )}
            <Nav.Bar.End style={{ paddingRight: "1rem" }}>
              {state.beta != null && <Beta.Tag />}
              <Button.Icon onClick={() => remove(key)} size="small">
                <Icon.Close style={{ color: "var(--pluto-gray-l8)" }} />
              </Button.Icon>
            </Nav.Bar.End>
          </Nav.Bar>
        )}
        <Content layoutKey={key} />
      </Core.Modal>
    </Menu.ContextMenu>
  );
};

export const Modals = (): ReactElement => {
  const layouts = useSelectModals();
  const remove = useRemover();
  return (
    <>
      {layouts.map((l) => (
        <Modal key={l.key} state={l} remove={remove} />
      ))}
    </>
  );
};
