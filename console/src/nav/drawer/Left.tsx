import { TimeSpan, useDebouncedCallback } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Drawer } from "@/nav/drawer/Drawer";
import { Item } from "@/nav/item";
import { Session } from "@/nav/session";

export interface LeftProps {}

export const Left = (): ReactElement => {
  const { selected, hover, size } = Session.useSelectLeft();
  const items = Item.useLeft();
  const dispatch = useDispatch();
  const activeItem = useMemo(() => {
    const item = items.find((i) => i.key === selected);
    if (item == null) return undefined;
    return size != null ? { ...item, initialSize: size } : item;
  }, [items, selected, size]);
  const onResize = useDebouncedCallback(
    (size: number) => dispatch(Session.resizeLeft({ size })),
    TimeSpan.milliseconds(100),
    [dispatch],
  );
  const onCollapse = useCallback(() => {
    if (hover) dispatch(Session.stopLeftHover({}));
    else if (selected != null) dispatch(Session.selectLeft({ key: selected }));
  }, [dispatch, hover, selected]);
  const onStopHover = useCallback(
    () => dispatch(Session.stopLeftHover({})),
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
