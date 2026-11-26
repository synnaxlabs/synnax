// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";

interface ScrollProps extends Omit<
  List.ScrollProps,
  "onKeyDown" | "onKeyUp" | "onBlur"
> {}

/**
 * Scroll is a focusable list scroll whose keys move the selection of the enclosing
 * `Select.Frame`, whose cursor must be `"selection"`.
 */
export const Scroll = ({ className, ...rest }: ScrollProps): ReactElement => {
  const { onKeyDown, onKeyUp, onBlur } = Select.useKeyboard();
  return (
    <List.Scroll
      tabIndex={0}
      className={CSS.cls(CSS.BE("schematic", "list"), className)}
      {...rest}
      onKeyDown={onKeyDown}
      onKeyUp={onKeyUp}
      onBlur={onBlur}
    />
  );
};
