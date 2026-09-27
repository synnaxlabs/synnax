// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode, useEffect } from "react";

import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type Input } from "@/input";
import { Text as InputText } from "@/input/Text";
import { useSearchContext } from "@/select/registry";
import { useClosed } from "@/select/scope";

/** Props for {@link Search}. */
export interface SearchProps {
  placeholder?: string;
  /** Called as the user types, to filter the frame's data. */
  onSearch?: (term: string) => void;
  /** Buttons rendered after the field. */
  actions?: Input.TextProps["children"];
  loading?: boolean;
}

/**
 * The search field of a selection dialog. It hides the fixed {@link Item}s whose text
 * does not match, and passes the term to `onSearch` for the frame's data. The term
 * clears when the dialog closes.
 */
export const Search = ({
  placeholder = "Search...",
  onSearch,
  actions,
  loading = false,
}: SearchProps): ReactNode => {
  const closed = useClosed();
  const { term, setTerm } = useSearchContext("Select.Search");
  const { variant } = Dialog.useContext();
  useEffect(() => () => setTerm(""), [setTerm]);
  if (closed) return null;
  const input = (
    <InputText
      value={term}
      autoFocus
      flush
      startContent={loading ? <Icon.Loading /> : <Icon.Search />}
      placeholder={placeholder}
      size={variant === "modal" ? "large" : "medium"}
      rounded
      grow
      full="x"
      onChange={(v) => {
        setTerm(v);
        onSearch?.(v);
      }}
    />
  );
  if (actions == null) return input;
  return (
    <Flex.Box pack x>
      {input}
      {actions}
    </Flex.Box>
  );
};
