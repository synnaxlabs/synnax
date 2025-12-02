// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/Frame.css";

import { type ontology } from "@synnaxlabs/client";
import {
  Flex,
  type Flux,
  List,
  Select,
  type state,
  useInactivity,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { i } from "node_modules/vite/dist/node/chunks/moduleRunnerTransport";
import {
  type PropsWithChildren,
  Provider,
  type ReactElement,
  use,
  useCallback,
  useState,
} from "react";

import { CSS } from "@/css";
import { type Request } from "@/view/View";

export interface FrameProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
> extends PropsWithChildren,
    Pick<Flux.UseListReturn<R, K, E>, "data" | "getItem" | "subscribe" | "retrieve"> {
  initialRequest: R;
  resourceType: ontology.ResourceType;
}

export const Frame = <
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
>({
  children,
  initialRequest,
  resourceType,
  data,
  getItem,
  subscribe,
  retrieve,
}: FrameProps<K, E, R>): ReactElement => {
  const [request, setRequest] = useState<R>(initialRequest);
  const [selected, setSelected] = useState<K[]>([]);
  const { ref } = useInactivity<HTMLDivElement>(500);
  const handleRequestChange = useCallback(
    (setter: state.SetArg<R>, opts?: Flux.AsyncListOptions) => {
      if (typeof setter === "function")
        retrieve((p) => setter({ ...request, ...p }), opts);
      else retrieve(setter, opts);
      setRequest(setter);
    },
    [retrieve],
  );
  const handleFetchMore = useCallback(
    () =>
      handleRequestChange((p) => ({ ...p, ...List.page(p, 25) }), { mode: "append" }),
    [handleRequestChange],
  );
  return (
    <Flex.Box full="y" empty className={CSS.BE("view", "frame")} ref={ref}>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={handleFetchMore}
      >
        {children}
      </Select.Frame>
    </Flex.Box>
  );
};

interface ContextValue<R extends Request> {
  request: R;
  resourceType: ontology.ResourceType;
}

const createViewContext = <R extends Request>(
  initialRequest: R,
  resourceType: ontology.ResourceType,
): ContextValue<R> => ({ request: initialRequest, resourceType });
