// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Ranger, Synnax } from "@synnaxlabs/pluto";

import { Palette } from "@/platform/palette";
import { Panel } from "@/platform/panel";
import { type Search } from "@/platform/search";
import { Session } from "@/session";

const SearchListItem: Search.ListItem = (props) => {
  const item = List.useItem<string, resource.Resource>(props.itemKey);
  const range = Ranger.useResult(item == null ? null : { key: item.id.key }).data;
  const client = Synnax.use();
  const store = Session.useStore();
  const openTab = Panel.useOpenTab();
  const handleError = Status.useErrorHandler();
  const handleSelect = () => {
    handleError(
      async () => {
        if (client == null || item == null) return;
        const ranges = await client.ranges.retrieve([item.id.key]);
        store.dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
        openTab({ variant: "resource", resource: ranger.resourceID(ranges[0].key) });
      },
      { message: "Failed to select range", name: item?.name ?? "" },
    );
  };
  if (item == null) return null;
  return (
    <Palette.ListItem gap="tiny" justify="between" {...props} onSelect={handleSelect}>
      <Text.Text weight={450} gap="medium">
        <Icon.Range />
        {item.name}
      </Text.Text>
      {range != null && (
        <Telem.Text.TimeRange level="tiny">{range.timeRange}</Telem.Text.TimeRange>
      )}
    </Palette.ListItem>
  );
};

export const SEARCH_LIST_ITEMS: Search.ListItems = { range: SearchListItem };
