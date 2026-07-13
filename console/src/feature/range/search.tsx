// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon, List, Status, Synnax, Telem, Text } from "@synnaxlabs/pluto";
import { type CrudeTimeRange } from "@synnaxlabs/x";

import { Layout } from "@/platform/layout";
import { Palette } from "@/platform/palette";
import { Range } from "@/platform/range";
import { type Search } from "@/platform/search";
import { Session } from "@/session";

const SearchListItem: Search.ListItem = (props) => {
  const resource = List.useItem<string, ontology.Resource>(props.itemKey);
  const client = Synnax.use();
  const store = Session.useStore();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  const handleSelect = () => {
    const name = resource?.name ?? "range";
    handleError(async () => {
      if (client == null || resource == null) return;
      const { id } = resource;
      const ranges = await client.ranges.retrieve([id.key]);
      store.dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
      const first = ranges[0];
      placeLayout({ ...Range.OVERVIEW_LAYOUT, name: first.name, key: first.key });
    }, `Failed to select ${name}`);
  };
  if (resource == null) return null;
  return (
    <Palette.ListItem gap="tiny" justify="between" {...props} onSelect={handleSelect}>
      <Text.Text weight={450} gap="medium">
        <Icon.Range />
        {resource?.name}
      </Text.Text>
      <Telem.Text.TimeRange level="small">
        {resource?.data?.timeRange as CrudeTimeRange}
      </Telem.Text.TimeRange>
    </Palette.ListItem>
  );
};

export const SEARCH_LIST_ITEMS: Search.ListItems = { range: SearchListItem };
