// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import type { PropsWithChildren, ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Filter } from "@/platform/view/filter";

export interface ToolbarProps<W> extends PropsWithChildren {
  /** The properties that the filter menu offers and the chips show. */
  properties: Filter.Properties<W>;
}

/**
 * The filter bar of a view: the filter menu and its chips, then the children, such as
 * sort and create, at the trailing edge.
 */
export const Toolbar = <W extends { and?: W[] }>({
  properties,
  children,
}: ToolbarProps<W>): ReactElement => (
  <Flex.Box x wrap align="center" background={1} className={CSS.BE("view", "toolbar")}>
    <Filter.Menu properties={properties} />
    <div className={CSS.BE("view", "chips")}>
      <Filter.Chips properties={properties} />
    </div>
    <Flex.Box x align="center" gap="small" className={CSS.BE("view", "actions")}>
      {children}
    </Flex.Box>
  </Flex.Box>
);
