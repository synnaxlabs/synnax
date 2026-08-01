// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { Flex, Form, Icon, Label, Tag } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { View } from "@/platform/view";

export const Chips = (): ReactElement | null => {
  const { editable } = View.useContext();
  const field = Form.useField<label.Key[]>("query.hasLabels", {
    optional: true,
  });
  const hasLabels = field?.value;
  const labels = Label.useRetrieveMultiple({ keys: hasLabels ?? [] }).data ?? [];
  if (labels.length === 0 || field == null || hasLabels == null) return null;
  const handleClose = (key: label.Key) =>
    field.onChange(hasLabels.filter((l) => l !== key));
  return (
    <Flex.Box x pack background={0}>
      <View.FilterChip>
        <Icon.Label />
        Labels
      </View.FilterChip>
      {labels.map(({ color, key, name }) => (
        <Tag.Tag
          key={key}
          color={color}
          size="small"
          onClose={editable ? () => handleClose(key) : undefined}
        >
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};
