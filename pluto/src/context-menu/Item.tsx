// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@/button";
import { Divider } from "@/context-menu/Divider";

export interface ItemProps
  extends Omit<Button.ButtonProps, "children" | "onClick">,
    Required<Pick<Button.ButtonProps, "children" | "onClick">> {
  showBottomDivider?: boolean;
}

export const Item = ({ showBottomDivider = false, ...props }: ItemProps) => (
  <>
    <Button.Button
      level="small"
      overflow="nowrap"
      variant="text"
      background={1}
      gap="small"
      {...props}
    />
    {showBottomDivider && <Divider />}
  </>
);
