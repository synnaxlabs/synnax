// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Arc } from "@/feature/arc";
import { Channel } from "@/feature/channel";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Range } from "@/feature/range";
import { Schematic } from "@/feature/schematic";
import { Search } from "@/feature/search";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { type Palette } from "@/platform/palette";

const SEARCH_LIST_ITEMS: Search.ListItems = {
  ...Arc.SEARCH_LIST_ITEMS,
  ...Channel.SEARCH_LIST_ITEMS,
  ...LinePlot.SEARCH_LIST_ITEMS,
  ...Log.SEARCH_LIST_ITEMS,
  ...Project.SEARCH_LIST_ITEMS,
  ...Range.SEARCH_LIST_ITEMS,
  ...Schematic.SEARCH_LIST_ITEMS,
  ...Table.SEARCH_LIST_ITEMS,
  ...Task.SEARCH_LIST_ITEMS,
};

export const List = (props: Palette.ListProps<ontology.Resource>): ReactElement => (
  <Search.List items={SEARCH_LIST_ITEMS} {...props} />
);
