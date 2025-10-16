// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import {
  Dialog,
  Flex,
  Icon,
  Label as PLabel,
  state,
  Tag,
  Text,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";

import { Label } from "@/label";

export interface SelectFiltersProps {
  request: ranger.RetrieveRequest;
  onRequestChange: state.Setter<ranger.RetrieveRequest>;
}

const FilterContextMenu = ({ request, onRequestChange }: SelectFiltersProps) => {
  const handleRequestChange = (setter: state.SetArg<ranger.RetrieveRequest>) => {
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

export const SelectFilters = ({ request, onRequestChange }: SelectFiltersProps) => (
  <Dialog.Frame>
    <Dialog.Trigger hideCaret>
      <Icon.Filter />
      <Text.Text>Filter</Text.Text>
    </Dialog.Trigger>
    <Dialog.Dialog
      background={1}
      style={{ padding: "1rem" }}
      borderColor={5}
      pack={false}
    >
      <FilterContextMenu request={request} onRequestChange={onRequestChange} />
    </Dialog.Dialog>
  </Dialog.Frame>
);

interface HasLabelsFilterProps {
  request: ranger.RetrieveRequest;
}

const HasLabelsFilter = ({ request }: HasLabelsFilterProps) => {
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

export const Filters = ({ request }: SelectFiltersProps) => (
  <Flex.Box x>
    <HasLabelsFilter request={request} />
  </Flex.Box>
);
