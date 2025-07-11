// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { Input } from "@/input";

export interface SearchInputProps {
  searchPlaceholder?: string;
  onSearch?: (term: string) => void;
}

export const SearchInput = ({
  searchPlaceholder = "Search...",
  onSearch,
}: SearchInputProps) => {
  const [term, setTerm] = useState<string>("");
  return (
    <Input.Text
      value={term}
      autoFocus
      placeholder={searchPlaceholder}
      borderShade={6}
      onChange={(v) => {
        setTerm(v);
        onSearch?.(v);
      }}
    />
  );
};
