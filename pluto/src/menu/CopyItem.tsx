// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useContext } from "@/menu/Menu";

export interface CopyItemProps extends Button.CopyProps {
  itemKey: string;
}

export const CopyItem = ({
  className,
  itemKey,
  ...rest
}: CopyItemProps): ReactElement => {
  const { onClick: ctxOnClick, level = "p", gap, background } = useContext();
  const handleClick: Button.ButtonProps["onClick"] = () => ctxOnClick(itemKey);
  return (
    <Button.Copy
      level={level}
      overflow="nowrap"
      variant="text"
      className={CSS(CSS.B("menu-item"), className)}
      background={background}
      tooltip={null}
      gap={gap}
      onClick={handleClick}
      propagateClick
      {...rest}
    />
  );
};
