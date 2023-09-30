// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { type Ontology } from "@/ontology";
import { type Range } from "@/range/range";
import { add } from "@/range/slice";

const handleSelect: Ontology.HandleSelect = ({ selection, client, store }) => {
  void (async () => {
    const ranges = await client.ranges.retrieve(selection.map((s) => s.id.key));
    const wsRanges: Range[] = ranges.map((r) => ({
      variant: "static",
      key: r.key,
      name: r.name,
      timeRange: {
        start: r.timeRange.start.valueOf(),
        end: r.timeRange.end.valueOf(),
      },
    }));
    store.dispatch(add({ ranges: wsRanges }));
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "range",
  hasChildren: false,
  icon: <Icon.Range />,
  canDrop: () => true,
  onSelect: handleSelect,
  haulItems: () => [],
  allowRename: () => true,
};
