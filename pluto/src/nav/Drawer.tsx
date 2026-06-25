// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/nav/Drawer.css";

import { type box, location } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Resize } from "@/resize";
import { Eraser } from "@/vis/eraser";

export interface DrawerProps extends Resize.SingleProps {}

export const Drawer = ({
  children,
  location: loc,
  collapseThreshold = 0.65,
  className,
  onResize,
  onResizeEnd,
  onCollapse,
  ...rest
}: DrawerProps): ReactElement => {
  eraseEnabled ??= open;
  const { erase } = Eraser.use({ enabled: eraseEnabled });
  const handleResize = useCallback(
    (size: number, box: box.Box) => {
      erase(box);
      onResize?.(size, box);
    },
    [erase, onResize],
  );
  const handleResizeEnd = useCallback(
    (size: number, box: box.Box) => {
      erase(box);
      onResizeEnd?.(size, box);
    }
    [erase, onResizeEnd],
  );
  return (
    <Resize.Single
      className={CSS(
        CSS.B("nav-drawer"),
        CSS.dir(location.direction(loc)),
        CSS.visible(open),
        className,
      )}
      collapseThreshold={collapseThreshold}
      location={loc}
      onResize={handleResize}
      onResizeEnd={handleResizeEnd}
      {...rest}
    >
      <Errors.Boundary>{children}</Errors.Boundary>
    </Resize.Single>
  );
};
