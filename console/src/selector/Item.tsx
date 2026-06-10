// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Button, type Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Layout } from "@/layout";
import { type Selectable } from "@/selector/Selector";

export interface ItemProps extends Omit<Button.ButtonProps, "children"> {
  title: string;
  icon: Icon.ReactElement;
}

export const Item = ({ title, icon, ...rest }: ItemProps) => (
  <Button.Button variant="outlined" {...rest}>
    {icon}
    {title}
  </Button.Button>
);

export interface SimpleItemProps {
  title: string;
  icon: Icon.ReactElement;
  layout: Layout.BaseState;
  useVisible?: () => boolean;
}

export const createSimpleItem = ({
  title,
  icon,
  layout,
  useVisible,
}: SimpleItemProps): Selectable => {
  const C: Selectable = ({ onPlace, onResolved }) => {
    const visible = useVisible?.() ?? true;
    // In a panel the item resolves into the tab as an inline view (the layout's type
    // selects the renderer, its args/name seed the view); otherwise it places the
    // layout as before (e.g. a window form).
    const handleClick = useCallback(() => {
      if (onResolved != null)
        onResolved({
          view: {
            type: layout.type,
            name: layout.name,
            args: layout.args as panel.TabView["args"],
          },
        });
      else onPlace(layout);
    }, [onPlace, onResolved]);
    if (!visible) return null;
    return <Item title={title} icon={icon} onClick={handleClick} />;
  };
  C.type = layout.type;
  C.useVisible = useVisible;
  return C;
};
