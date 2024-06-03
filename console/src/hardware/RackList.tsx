// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Synnax } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement } from "react";

export const RackList = (): ReactElement => {
  const client = Synnax.use();

  return (
    <List.List<number, rack.Rack>>
      <List.Search searcher={client?.hardware.racks}>
        {(p) => (
          <Input.Text
            size="large"
            placeholder={
              <Text.WithIcon level="p" startIcon={<Icon.Search />}>
                Search racks
              </Text.WithIcon>
            }
            {...p}
          />
        )}
      </List.Search>
      <List.Core<number, rack.Rack>>{(p) => <RackListItem {...p} />}</List.Core>
    </List.List>
  );
};

interface RackListItemProps extends List.ItemProps<number, rack.Rack> {}

const RackListItem = (props: RackListItemProps): ReactElement => {
  const {
    entry: { name },
  } = props;
  return (
    <List.ItemFrame {...props}>
      <Text.Text level="h5" weight={500}>
        {name}
      </Text.Text>
    </List.ItemFrame>
  );
};
