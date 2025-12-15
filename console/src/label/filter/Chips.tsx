// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/label/filter/Chips.css";

import { type label } from "@synnaxlabs/client";
import { Flex, Form, Icon, Label, Tag, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { View } from "@/view";

export const Chips = (): ReactElement | null => {
  const { editable } = View.useContext();
  const field = Form.useField<label.Key[]>("query.hasLabels", {
    optional: true,
  });
  const hasLabels = Form.useFieldValue<label.Key[]>("query.hasLabels", {
    optional: true,
  });
  const labels = Label.useRetrieveMultiple({ keys: hasLabels ?? [] }).data ?? [];
  if (labels.length === 0 || field == null || hasLabels == null) return null;
  const handleClose = (key: label.Key) =>
    field.onChange(hasLabels.filter((l) => l !== key));
  return (
    <Flex.Box x pack background={0}>
      <Text.Text
        bordered
        className={CSS.BE("label", "filter-chips")}
        size="small"
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
          onClose={editable ? () => handleClose(key) : undefined}
        >
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};
