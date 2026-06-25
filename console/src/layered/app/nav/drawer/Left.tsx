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
import { Drawer } from "@/layered/service/nav/Drawer";
import { Session } from "@/layered/session";

export const Left = (): ReactElement => {
  const { selected, hover, size } = Session.Nav.useSelectLeft();
  const dispatch = useDispatch();
  const item = Items.LEFT.find((i) => i.key === selected);
  const onResize = useCallback(
    (size: number) => dispatch(Session.Nav.resizeLeft({ size })),
    [dispatch],
  );
  const onCollapse = useCallback(() => {
    if (hover) dispatch(Session.Nav.stopLeftHover({}));
    else if (selected != null) dispatch(Session.Nav.selectLeft({ key: selected }));
  }, [dispatch, hover, selected]);
  const onStopHover = useCallback(
    () => dispatch(Session.Nav.stopLeftHover({})),
    [dispatch],
  );
  return (
    <Drawer
      location="left"
      open={item != null}
      size={size ?? item?.initialSize ?? Items.DEFAULT_SIZE}
      sizeBounds={{ lower: item?.minSize, upper: item?.maxSize }}
      hover={hover}
      onResize={onResize}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    >
      {item?.content}
    </Drawer>
  );
};
