// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { type Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type Input } from "@/input";
import { Text as InputText } from "@/input/Text";

/** Props for {@link SearchInput}. */
export interface SearchInputProps {
  searchPlaceholder?: string;
  /** Called as the user types. Leave it unset to hide the input. */
  onSearch?: (term: string) => void;
  /** Buttons rendered inside the input, after the term. */
  actions?: Input.TextProps["children"];
  dialogVariant?: Dialog.FrameProps["variant"];
  loading?: boolean;
}

/** The search field at the top of a select {@link Dialog}. */
export const SearchInput = ({
  searchPlaceholder = "Search...",
  onSearch,
  actions,
  dialogVariant = "floating",
  loading = false,
}: SearchInputProps) => {
  const [term, setTerm] = useState<string>("");
  const inputContent = (
    <InputText
      value={term}
      autoFocus
      flush
      startContent={loading ? <Icon.Loading /> : <Icon.Search />}
      placeholder={searchPlaceholder}
      size={dialogVariant === "modal" ? "large" : "medium"}
      rounded
      grow
      full="x"
      onChange={(v) => {
        setTerm(v);
        onSearch?.(v);
      }}
    />
  );
  if (actions == null) return inputContent;
  return (
    <Flex.Box pack x>
      {inputContent}
      {actions}
    </Flex.Box>
  );
};
