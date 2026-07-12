// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, type Icon } from "@synnaxlabs/pluto";

import { type Selectable } from "@/platform/selector/Selector";

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

export interface CreateSelectableConfig {
  type: string;
  title: string;
  icon: Icon.ReactElement;
  useOnSelect: () => () => void;
  useVisible?: () => boolean;
}

export const createSelectable = ({
  title,
  icon,
  type,
  useOnSelect,
  useVisible,
}: CreateSelectableConfig): Selectable => {
  const C: Selectable = () => {
    const visible = useVisible?.() ?? true;
    const onSelect = useOnSelect();
    if (!visible) return null;
    return <Item title={title} icon={icon} onClick={() => onSelect()} />;
  };
  C.type = type;
  C.useVisible = useVisible;
  return C;
};
