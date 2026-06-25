// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Items } from "@/layered/app/nav/items";
import { Service } from "@/layered/service";
import { Session } from "@/layered/session";

export const Bottom = (): ReactElement | null => {
  const { visible, hover, size } = Session.Nav.useSelectBottom();
  const dispatch = useDispatch();
  const onResizeEnd = useCallback(
    (size: number) => dispatch(Session.Nav.resizeBottom({ size })),
    [dispatch],
  );
  const onCollapse = useCallback(() => {
    if (hover) dispatch(Session.Nav.stopBottomHover({}));
    else dispatch(Session.Nav.selectBottom({}));
  }, [dispatch, hover]);
  const onStopHover = useCallback(
    () => dispatch(Session.Nav.stopBottomHover({})),
    [dispatch],
  );
  const { initialSize = Items.DEFAULT_SIZE, sizeBounds, content } = Items.BOTTOM;
  return (
    <Service.Nav.Drawer
      location="bottom"
      size={size ?? initialSize}
      sizeBounds={sizeBounds}
      hover={hover}
      visible={visible}
      onResizeEnd={onResizeEnd}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    >
      {content}
    </Service.Nav.Drawer>
  );
};
