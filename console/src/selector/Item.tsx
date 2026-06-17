// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Button, type Icon, Panel as Base } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { type Selectable } from "@/selector/Selector";
import { type Tabs } from "@/tabs";

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
  useVisible?: () => boolean;
  type: string;
}

export const createSimpleItem = ({
  title,
  icon,
  useVisible,
  type,
}: SimpleItemProps): Selectable => {
  const C: Selectable = ({ tabKey }) => {
    const visible = useVisible?.() ?? true;
    const dispatch = Base.useSingleDispatch();
    const handleClick = useCallback(() => {
      dispatch([
        panel.setTabType({ key: tabKey, type }),
        panel.setTabArgs({ key: tabKey, args: { name: title } }),
      ]);
    }, [dispatch]);
    if (!visible) return null;
    return <Item title={title} icon={icon} onClick={handleClick} />;
  };
  C.type = type;
  C.useVisible = useVisible;
  return C;
};
