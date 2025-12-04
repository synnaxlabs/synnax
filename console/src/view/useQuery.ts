// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, type List, type state } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback, useMemo, useState } from "react";

export interface Query extends List.PagerParams, record.Unknown {}

export interface UseQueryReturn<Q extends Query = Query> {
  query: Q;
  onQueryChange: (setter: state.SetArg<Q>, opts?: Flux.AsyncListOptions) => void;
}

export const useQuery = <Q extends Query>(
  initialQuery: Q,
  retrieve: (query: state.SetArg<Q, Partial<Q>>, opts?: Flux.AsyncListOptions) => void,
): UseQueryReturn<Q> => {
  const [query, setQuery] = useState(initialQuery);
  const handleQueryChange = useCallback(
    (setter: state.SetArg<Q>, opts?: Flux.AsyncListOptions) => {
      if (typeof setter === "function")
        retrieve((p) => setter({ ...query, ...p }), opts);
      else retrieve(setter, opts);
      setQuery(setter);
    },
    [retrieve, query],
  );
  return useMemo(
    () => ({ query, onQueryChange: handleQueryChange }),
    [query, handleQueryChange],
  );
};
