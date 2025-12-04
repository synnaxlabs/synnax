// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

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
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useState,
} from "react";

import { CSS } from "@/css";
import { Provider, type Request } from "@/view/context";

export interface FrameProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
> extends PropsWithChildren,
    Pick<Flux.UseListReturn<R, K, E>, "data" | "getItem" | "subscribe"> {
  onRequestChange: (setter: state.SetArg<R>, opts?: Flux.AsyncListOptions) => void;
  resourceType: ontology.ResourceType;
}

export const Frame = <
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
>({
  children,
  onRequestChange,
  resourceType,
  data,
  getItem,
  subscribe,
}: FrameProps<K, E, R>): ReactElement => {
  const [selected, setSelected] = useState<K[]>([]);
  const [editable, setEditable] = useState(true);
  const { visible, ref } = useInactivity<HTMLDivElement>(500);

  const handleFetchMore = useCallback(
    () => onRequestChange((p) => ({ ...p, ...List.page(p, 25) }), { mode: "append" }),
    [onRequestChange],
  );

  return (
    <Flex.Box full="y" empty className={CSS.B("view")} ref={ref}>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={handleFetchMore}
      >
        <Provider value={{ editable, setEditable, visible, resourceType }}>
          {children}
        </Provider>
      </Select.Frame>
    </Flex.Box>
  );
};
