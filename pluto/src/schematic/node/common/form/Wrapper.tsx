// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/form.css";

import { createContext, type FC, type ReactElement, useContext } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Text } from "@/text";

/** StyleLockContext marks the style fields below it as locked while grouped. */
export const StyleLockContext = createContext(false);
StyleLockContext.displayName = "StyleLockContext";

interface WrapperProps extends Flex.BoxProps {
  /** lockable marks style fields, replaced by a notice under StyleLockContext. */
  lockable?: boolean;
}

export const Wrapper: FC<WrapperProps> = ({
  className,
  direction,
  lockable = false,
  ...rest
}): ReactElement => {
  const locked = useContext(StyleLockContext) && lockable;
  if (locked)
    return (
      <Text.Text status="disabled" center>
        Style editing disabled while grouped. Ungroup to edit.
      </Text.Text>
    );
  return (
    <Flex.Box
      direction={direction}
      align="stretch"
      className={CSS.cls(CSS.B("symbol-form"), className)}
      gap={direction === "x" ? "large" : "medium"}
      {...rest}
    />
  );
};
