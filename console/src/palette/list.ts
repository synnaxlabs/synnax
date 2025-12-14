// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component, type List, type state } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

export interface RetrieveParams extends List.PagerParams {
  term?: string;
  offset?: number;
  limit?: number;
}

export interface UseListReturn<E extends record.Keyed<string>> extends Pick<
  List.FrameProps<string, E>,
  "getItem" | "subscribe"
> {
  data: string[];
  handleSelect: (key: string) => void;
  listItem: Component.RenderProp<List.ItemProps<string>>;
  retrieve: (query: state.SetArg<RetrieveParams>) => void;
}
