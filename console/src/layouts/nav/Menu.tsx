// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { CSS as PCSS, Menu as PMenu, Text } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

export interface MenuProps extends Omit<PMenu.MenuProps, "children" | "onChange"> {
  children: Layout.NavMenuItem[];
  activeItem?: Layout.NavDrawerItem;
  onChange: (key: string, hover?: boolean) => void;
  onStartHover: (key: string) => void;
  onStopHover: () => void;
}

export const Menu = ({
  children,
  activeItem,
  onChange,
  onStartHover,
  onStopHover,
  ...rest
}: MenuProps): ReactElement => {
  const positionRef = useRef<xy.XY>({ ...xy.ZERO });
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  return (
    <PMenu.Menu {...rest} onChange={onChange}>
      {children.map(({ key, tooltip, icon }) => (
        <PMenu.Item.Icon
          className={CSS(
            CSS.BE("main-nav", "item"),
            PCSS.selected(activeItem?.key === key),
          )}
          onClick={() => {
            if (timeoutRef.current != null) {
              clearTimeout(timeoutRef.current);
              timeoutRef.current = null;
            }
          }}
          onMouseEnter={(e) => {
            timeoutRef.current = setTimeout(() => {
              timeoutRef.current = null;
              onStartHover(key);
              positionRef.current = xy.construct(e);
              const lis = (e: MouseEvent) => {
                const delta = xy.translation(xy.construct(e), positionRef.current);
                if (delta.y > 20 && Math.abs(delta.x) < 20) {
                  onStopHover();
                  window.removeEventListener("mousemove", lis);
                }
              };
              window.addEventListener("mousemove", lis);
            }, 250);
          }}
          onMouseLeave={() => {
            if (timeoutRef.current != null) {
              clearTimeout(timeoutRef.current);
              timeoutRef.current = null;
            }
          }}
          key={key}
          itemKey={key}
          size="large"
          tooltip={<Text.Text level="small">{tooltip}</Text.Text>}
          shade={2}
        >
          {icon}
        </PMenu.Item.Icon>
      ))}
    </PMenu.Menu>
  );
};
