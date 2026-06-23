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
import { Session } from "@/layered/session";
import { View } from "@/layered/view";

export const Bottom = (): ReactElement => {
  const { visible, hover, size } = Session.Nav.useSelectBottom();
  const dispatch = useDispatch();
  const onResize = useCallback(
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
  return (
    <View.Nav.Drawer
      location="bottom"
      open={visible}
      size={size ?? Items.BOTTOM.initialSize ?? Items.DEFAULT_SIZE}
      sizeBounds={{ lower: Items.BOTTOM.minSize, upper: Items.BOTTOM.maxSize }}
      hover={hover}
      onResize={onResize}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    >
      {Items.BOTTOM.content}
    </View.Nav.Drawer>
  );
};
