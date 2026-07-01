// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { Items } from "@/app/nav/items";
import { Feature } from "@/feature";
import { Session } from "@/session";

export const Bottom = (): ReactElement | null => {
  const { visible, hover, size } = Session.Nav.useSelectBottom();
  const dispatch = Session.useDispatch();
  const handleResizeEnd = useCallback(
    (size: number) => dispatch(Session.Nav.resizeBottom({ size })),
    [dispatch],
  );
  const handleCollapse = useCallback(
    () => dispatch(Session.Nav.collapseBottom({})),
    [dispatch],
  );
  const handleStopHover = useCallback(
    () => dispatch(Session.Nav.stopBottomHover({})),
    [dispatch],
  );
  const { initialSize = Items.DEFAULT_SIZE, sizeBounds, content } = Items.BOTTOM;
  return (
    <Feature.Nav.Drawer
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
    </Feature.Nav.Drawer>
  );
};
