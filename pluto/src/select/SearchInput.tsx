// Copyright 2025 Synnax Labs, Inc.
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
import { type Input } from "@/input";
import { Text as InputText } from "@/input/Text";

export interface SearchInputProps {
  searchPlaceholder?: string;
  onSearch?: (term: string) => void;
  actions?: Input.TextProps["children"];
  dialogVariant?: Dialog.FrameProps["variant"];
}

export const SearchInput = ({
  searchPlaceholder = "Search...",
  onSearch,
  actions,
  dialogVariant = "floating",
}: SearchInputProps) => {
  const [term, setTerm] = useState<string>("");
  const inputContent = (
    <InputText
      value={term}
      autoFocus
      placeholder={searchPlaceholder}
      size={dialogVariant === "modal" ? "large" : "medium"}
      contrast={3}
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
