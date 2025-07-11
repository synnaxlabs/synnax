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
import { type state } from "@/state";

export interface SearchParams {
  term?: string;
}

export interface SearchInputProps<P extends SearchParams> {
  searchPlaceholder?: string;
  onSearch?: state.Setter<P, P | {}>;
}

export const SearchInput = <P extends SearchParams>({
  searchPlaceholder = "Search...",
  onSearch,
}: SearchInputProps<P>) => {
  const [term, setTerm] = useState<string>("");
  return (
    <Input.Text
      value={term}
      autoFocus
      placeholder={searchPlaceholder}
      borderShade={6}
      onChange={(v) => {
        setTerm(v);
        onSearch?.((prev) => ({ ...prev, term: v }) as P);
      }}
    />
  );
};
