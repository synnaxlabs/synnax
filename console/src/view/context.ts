// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { context, type Flux, type List, type state } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

export interface Request extends List.PagerParams, record.Unknown {}

export interface ContextValue {
  editable: boolean;
  setEditable: state.Setter<boolean>;
  visible: boolean;
  resourceType: ontology.ResourceType;
}

export const [Provider, useContext] = context.create<ContextValue>({
  displayName: "Frame.Context",
  providerName: "Frame.Provider",
});

export interface UseRequestReturn<R extends Request> {
  request: R;
  onRequestChange: (setter: state.SetArg<R>, opts?: Flux.AsyncListOptions) => void;
}

export const useRequest = <R extends Request>(
  initialRequest: R,
  retrieve: (query: state.SetArg<R, Partial<R>>, opts?: Flux.AsyncListOptions) => void,
): UseRequestReturn<R> => {
  const [request, setRequest] = useState(initialRequest);
  const handleRequestChange = useCallback(
    (setter: state.SetArg<R>, opts?: Flux.AsyncListOptions) => {
      if (typeof setter === "function")
        retrieve((p) => setter({ ...request, ...p }), opts);
      else retrieve(setter, opts);
      setRequest(setter);
    },
    [retrieve, request],
  );
  return { request, onRequestChange: handleRequestChange };
};
