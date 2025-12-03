// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Flex, Icon, Label as PLabel, state, Tag, Text } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Label } from "@/label";
import { View } from "@/view";

export const FilterContextMenu = (): ReactElement => {
  const { request, onRequestChange } =
    View.useContext<status.MultiRetrieveArgs>("FilterContextMenu");

  const handleRequestChange = (setter: state.SetArg<status.MultiRetrieveArgs>) => {
    onRequestChange((prev) => {
      const next = state.executeSetter(setter, prev);
      return { ...next, offset: 0, limit: 0 };
    });
  };

  return (
    <Label.SelectMultiple
      value={request.hasLabels ?? []}
      onChange={(labels) => handleRequestChange((r) => ({ ...r, hasLabels: labels }))}
      triggerProps={{ hideTags: true, variant: "text" }}
      location={{ targetCorner: location.TOP_RIGHT, dialogCorner: location.TOP_LEFT }}
    />
  );
};

const HasLabelsFilter = (): ReactElement | null => {
  const { request } = View.useContext<status.MultiRetrieveArgs>("HasLabelsFilter");
  const labels =
    PLabel.useRetrieveMultiple({ keys: request.hasLabels ?? [] }).data ?? [];
  if (request.hasLabels == null || request.hasLabels.length === 0) return null;
  return (
    <Flex.Box x pack background={0}>
      <Text.Text
        el="span"
        bordered
        size="small"
        style={{ padding: "0 1rem", boxShadow: "var(--pluto-shadow-v1)" }}
        background={0}
        borderColor={5}
        level="small"
        color={9}
      >
        <Icon.Label />
        Labels
      </Text.Text>
      {labels.map(({ color, key, name }) => (
        <Tag.Tag key={key} color={color} size="small" textColor={9}>
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};

export const Filters = (): ReactElement | null => <HasLabelsFilter />;
