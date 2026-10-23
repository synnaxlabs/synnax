// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyboardEvent, type ReactElement } from "react";

import { Dialog } from "@/dialog";
import { Text, type TextProps } from "@/input/Text";

/** Props for {@link Search}. */
export interface SearchProps extends TextProps {}

/**
 * A field that filters the content of a dialog. It takes focus when it mounts. Escape
 * closes the enclosing {@link Dialog.Frame}, where a {@link Text} field only gives up
 * focus.
 */
export const Search = ({ onKeyDown, ...rest }: SearchProps): ReactElement => {
  const { close } = Dialog.useContext();
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    onKeyDown?.(e);
    if (e.key === "Escape") close();
  };
  return <Text autoFocus {...rest} onKeyDown={handleKeyDown} />;
};
