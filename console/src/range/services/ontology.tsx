// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type Haul, Icon, List, Select, Telem, Text } from "@synnaxlabs/pluto";
import { type CrudeTimeRange, strings } from "@synnaxlabs/x";

import { Ontology } from "@/ontology";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { add } from "@/range/slice";
import { fromClientRange } from "@/range/translate";

const handleSelect: Ontology.HandleSelect = ({
  selection,
  client,
  store,
  placeLayout,
  handleError,
}) => {
  const names = strings.naturalLanguageJoin(
    selection.map(({ name }) => name),
    "range",
  );
  handleError(async () => {
    const ranges = await client.ranges.retrieve(selection.map((s) => s.id.key));
    store.dispatch(add({ ranges: fromClientRange(ranges) }));
    const first = ranges[0];
    placeLayout({ ...OVERVIEW_LAYOUT, name: first.name, key: first.key });
  }, `Failed to select ${names}`);
};

const haulItems = ({ id }: ontology.Resource): Haul.Item[] => [
  { type: "range", key: id.key },
];

const PaletteListItem: Ontology.PaletteListItem = (props) => {
  const resource = List.useItem<string, ontology.Resource>(props.itemKey);
  return (
    <Select.ListItem gap="tiny" highlightHovered justify="between" {...props}>
      <Text.Text weight={450} gap="medium">
        <Icon.Range />
        {resource?.name}
      </Text.Text>
      <Telem.Text.TimeRange level="small">
        {resource?.data?.timeRange as CrudeTimeRange}
      </Telem.Text.TimeRange>
    </Select.ListItem>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "range",
  icon: <Icon.Range />,
  onSelect: handleSelect,
  canDrop: () => true,
  haulItems,
  PaletteListItem,
};
