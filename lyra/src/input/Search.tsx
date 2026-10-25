// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyboardEvent, type ReactElement } from "react";

import { Text, type TextProps } from "@/input/Text";

/** Props for {@link Search}. */
export interface SearchProps extends TextProps {}

/**
 * A field that filters the content of a dialog. It takes focus when it mounts. Escape
 * goes on to the dialog, which closes, or first closes a dialog open inside it. A
 * {@link Text} field keeps Escape and only gives up focus.
 */
export const Search = ({ onKeyDown, ...rest }: SearchProps): ReactElement => {
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    onKeyDown?.(e);
    if (e.key === "Escape") e.preventDefault();
  };
  return <Text autoFocus {...rest} onKeyDown={handleKeyDown} />;
};
