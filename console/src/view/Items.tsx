// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component, Flex, List, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement, type ReactNode } from "react";

import { useContext } from "@/view/context";

export interface ItemsProps<K extends record.Key> {
  children: Component.RenderProp<List.ItemProps<K>>;
  emptyContent?: ReactNode;
}

export const Items = <K extends record.Key>({
  children,
  emptyContent,
}: ItemsProps<K>): ReactElement => {
  const { resourceType } = useContext("Items");

  return (
    <List.Items<K>
      emptyContent={emptyContent ?? <DefaultEmptyContent resourceType={resourceType} />}
      grow
    >
      {children}
    </List.Items>
  );
};

interface DefaultEmptyContentProps {
  resourceType: string;
}

const DefaultEmptyContent = ({
  resourceType,
}: DefaultEmptyContentProps): ReactElement => (
  <Flex.Box center grow>
    <Text.Text level="p" status="disabled">
      No {plural(resourceType)} found.
    </Text.Text>
  </Flex.Box>
);
