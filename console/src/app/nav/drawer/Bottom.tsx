// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Toolbars } from "@/app/toolbars";
import { Nav } from "@/platform/nav";
import { Session } from "@/session";

export const Bottom = (): ReactElement | null => {
  const { visible, hover, size } = Session.Nav.useSelectBottom();
  const dispatch = Session.useDispatch();
  const handleResizeEnd = (size: number) =>
    dispatch(Session.Nav.resizeBottom({ size }));
  const handleCollapse = () => dispatch(Session.Nav.collapseBottom({}));
  const handleStopHover = () => dispatch(Session.Nav.stopBottomHover({}));
  const { initialSize = Toolbars.DEFAULT_SIZE, sizeBounds, content } = Toolbars.BOTTOM;
  return (
    <Nav.Drawer
      location="bottom"
      size={size ?? initialSize}
      sizeBounds={sizeBounds}
      hover={hover}
      collapsed={!visible}
      onResizeEnd={handleResizeEnd}
      onCollapse={handleCollapse}
      onStopHover={handleStopHover}
    >
      {content}
    </Nav.Drawer>
  );
};
