import { TimeSpan, useDebouncedCallback } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Drawer } from "@/nav/drawer/Drawer";
import { Item } from "@/nav/item";
import { Session } from "@/nav/session";

export const Bottom = (): ReactElement => {
  const { visible, hover, size } = Session.useSelectBottom();
  const item = Item.useBottom();
  const dispatch = useDispatch();
  const activeItem = useMemo(() => {
    if (!visible) return undefined;
    return size != null ? { ...item, initialSize: size } : item;
  }, [item, visible, size]);
  const onResize = useDebouncedCallback(
    (size: number) => dispatch(Session.resizeBottom({ size })),
    TimeSpan.milliseconds(100),
    [dispatch],
  );
  const onCollapse = useCallback(() => {
    if (hover) dispatch(Session.stopBottomHover({}));
    else dispatch(Session.selectBottom({}));
  }, [dispatch, hover]);
  const onStopHover = useCallback(
    () => dispatch(Session.stopBottomHover({})),
    [dispatch],
  );
  return (
    <Drawer
      location="bottom"
      activeItem={activeItem}
      hover={hover}
      onResize={onResize}
      onCollapse={onCollapse}
      onStopHover={onStopHover}
    />
  );
};
