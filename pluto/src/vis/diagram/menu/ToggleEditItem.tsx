// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { Item, type ItemProps } from "@/menu/Item";
import { useContext as useControlContext } from "@/telem/control/Controller";
import { useContext } from "@/vis/diagram/Context";

export interface ToggleEditItemProps extends Omit<
  ItemProps,
  "itemKey" | "onClick" | "children"
> {}

export const ToggleEditItem = ({
  disabled,
  ...rest
}: ToggleEditItemProps): ReactElement => {
  const { editable, onEditableChange } = useContext();
  const { status } = useControlContext();
  return (
    <Item
      itemKey="toggle-edit"
      onClick={() => onEditableChange(!editable)}
      disabled={disabled || status === "acquired"}
      {...rest}
    >
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
      {editable ? "Switch to View Mode" : "Switch to Edit Mode"}
    </Item>
  );
};
