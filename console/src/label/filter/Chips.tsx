// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { Flex, Form, Icon, Label, Tag, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { View } from "@/view";

export const Chips = (): ReactElement | null => {
  const { editable } = View.useContext();
  const { set } = Form.useContext();
  const hasLabels = Form.useFieldValue<label.Key[]>("query.hasLabels", {
    defaultValue: [],
  });
  const labels = Label.useRetrieveMultiple({ keys: hasLabels ?? [] }).data ?? [];
  if (labels.length === 0) return null;
  const handleClose = (key: label.Key) => {
    set(
      "query.hasLabels",
      hasLabels.filter((k) => k !== key),
      { notifyOnChange: true },
    );
  };
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
          onClose={editable ? () => handleClose(key) : undefined}
        >
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};
