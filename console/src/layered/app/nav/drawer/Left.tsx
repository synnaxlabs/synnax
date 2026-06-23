// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, useDebouncedCallback } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Items } from "@/layered/app/nav/items";
import { Session } from "@/layered/session";
import { Drawer } from "@/layered/view/nav/Drawer";

export const Left = (): ReactElement => {
  const { selected, hover, size } = Session.Nav.useSelectLeft();
  const dispatch = useDispatch();
  const activeItem = useMemo(() => {
    const item = Items.LEFT.find((i) => i.key === selected);
    if (item == null) return undefined;
    return size != null ? { ...item, initialSize: size } : item;
  }, [selected, size]);
  const onResize = useDebouncedCallback(
    (size: number) => dispatch(Session.Nav.resizeLeft({ size })),
    TimeSpan.milliseconds(100),
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
      activeItem={activeItem}
      hover={hover}
      onResize={onResize}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    />
  );
};
