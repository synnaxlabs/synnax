// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb, Button, Dialog, Icon, Nav } from "@synnaxlabs/pluto";
import { type CSSProperties, type ReactElement } from "react";

import { Content } from "@/modals/Content";
import { type Spec, type WindowProps } from "@/modals/Modals";

const layoutCSS = (window?: WindowProps): CSSProperties => ({
  maxWidth: window?.size?.width,
  maxHeight: window?.size?.height,
  minWidth: window?.minSize?.width,
  minHeight: window?.minSize?.height,
});

export interface ModalProps {
  spec: Spec;
  onClose: (result?: unknown) => void;
}

export const Modal = ({ spec, onClose }: ModalProps): ReactElement => {
  const { key, name, window, icon } = spec;
  return (
    <Dialog.Frame
      key={key}
      variant="modal"
      visible
      onVisibleChange={() => onClose()}
      background={0}
    >
      <Dialog.Dialog style={layoutCSS(window)} full>
        {window?.navTop && (
          <Nav.Bar location="top" size="6rem" bordered>
            {(window?.showTitle ?? true) && (
              <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
                <Breadcrumb.Breadcrumb gap="tiny">
                  <Breadcrumb.Segment color={9}>
                    {Icon.resolve(icon)}
                  </Breadcrumb.Segment>
                  {name.split(".").map((segment) => (
                    <Breadcrumb.Segment color={9} key={segment} weight={400}>
                      {segment}
                    </Breadcrumb.Segment>
                  ))}
                </Breadcrumb.Breadcrumb>
              </Nav.Bar.Start>
            )}
            <Nav.Bar.End style={{ paddingRight: "1rem" }}>
              <Button.Button
                onClick={() => onClose()}
                size="small"
                variant="text"
                textColor={9}
              >
                <Icon.Close />
              </Button.Button>
            </Nav.Bar.End>
          </Nav.Bar>
        )}
        <Content spec={spec} onClose={onClose} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
