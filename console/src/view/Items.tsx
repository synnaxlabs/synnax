// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, List, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement } from "react";

import { useContext } from "@/view/context";

export interface ItemsProps<K extends record.Key = record.Key>
  extends List.ItemsProps<K> {}

export const Items = <K extends record.Key>(props: ItemsProps<K>): ReactElement => (
  <List.Items<K> emptyContent={emptyContent} displayItems={Infinity} grow {...props} />
);

const DefaultEmptyContent = (): ReactElement => {
  const { resourceType } = useContext("View.Items");
  return (
    <Flex.Box center grow>
      <Text.Text status="disabled">No {plural(resourceType)} found.</Text.Text>
    </Flex.Box>
  );
};

const emptyContent = <DefaultEmptyContent />;
