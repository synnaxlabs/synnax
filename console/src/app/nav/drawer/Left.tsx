// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { Toolbars } from "@/app/toolbars";
import { Nav } from "@/platform/nav";
import { Session } from "@/session";

export const Left = (): ReactElement => {
  const { selected, hover, size } = Session.Nav.useSelectLeft();
  const dispatch = Session.useDispatch();
  const item = Toolbars.LEFT.find((i) => i.key === selected);
  const handleResizeEnd = useCallback(
    (size: number) => dispatch(Session.Nav.resizeLeft({ size })),
    [dispatch],
  );
  const handleCollapse = useCallback(
    () => dispatch(Session.Nav.collapseLeft({})),
    [dispatch],
  );
  const handleStopHover = useCallback(
    () => dispatch(Session.Nav.stopLeftHover({})),
    [dispatch],
  );
  const { initialSize = Toolbars.DEFAULT_SIZE, sizeBounds, content } = item ?? {};
  return (
    <Nav.Drawer
      location="left"
      size={size ?? initialSize}
      sizeBounds={sizeBounds}
      hover={hover}
      collapsed={item == null}
      onResizeEnd={handleResizeEnd}
      onCollapse={handleCollapse}
      onStopHover={handleStopHover}
    >
      {content}
    </Nav.Drawer>
  );
};
