// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useState } from "react";

import Fuse from "fuse.js";

export const useSearch = <E extends Record<string, unknown>>(): [
  string,
  (value: string) => void,
  (data: E[]) => E[]
] => {
  const [query, setQuery] = useState("");
  const searchFunc = useCallback(
    (data: E[]) => {
      if (data?.length === 0) return data;
      const fuse = new Fuse(data, { keys: Object.keys(data[0]) });
      return query.length > 0 ? fuse.search(query).map((res) => res.item) : data;
    },
    [query]
  );
  return [query, setQuery, searchFunc];
};
