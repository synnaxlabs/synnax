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
import { useContext } from "@/telem/control/Controller";

export interface ToggleItemProps extends Omit<
  ItemProps,
  "itemKey" | "onClick" | "children"
> {}

export const ToggleItem = (props: ToggleItemProps): ReactElement | null => {
  const { key, status, acquire, release } = useContext();
  if (key === "") return null;
  const acquired = status === "acquired";
  return (
    <Item
      itemKey="control-toggle"
      onClick={() => (acquired ? release() : acquire())}
      {...props}
    >
      <Icon.Control />
      {acquired ? "Release Control" : "Take Control"}
    </Item>
  );
};
