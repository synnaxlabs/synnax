// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { Flex, Icon, Label, Tag, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type HasQuery } from "@/label/filter/types";
import { type View } from "@/view";

export interface ChipsProps
  extends Pick<View.UseQueryReturn<HasQuery>, "query" | "onQueryChange"> {
  isClosable?: boolean;
}

export const Chips = ({
  query,
  onQueryChange,
  isClosable = false,
}: ChipsProps): ReactElement | null => {
  const { hasLabels } = query;
  const labels = Label.useRetrieveMultiple({ keys: hasLabels ?? [] }).data ?? [];
  if (labels.length === 0) return null;
  const handleClose = (key: label.Key) =>
    onQueryChange(({ hasLabels, ...rest }) => ({
      ...rest,
      hasLabels: hasLabels?.filter((k) => k !== key),
    }));
  return (
    <Flex.Box x pack background={0}>
      <Text.Text
        bordered
        size="small"
        style={{ padding: "0 1rem", boxShadow: "var(--pluto-shadow-v1)" }}
        borderColor={5}
        level="small"
      >
        <Icon.Label />
        Labels
      </Text.Text>
      {labels.map(({ color, key, name }) => (
        <Tag.Tag
          key={key}
          color={color}
          size="small"
          onClose={isClosable ? () => handleClose(key) : undefined}
        >
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};
