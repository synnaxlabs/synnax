// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/list/List.css";

import { type label, type ranger } from "@synnaxlabs/client";
import {
  Dialog,
  Flex,
  Icon,
  Label as PLabel,
  Menu,
  Tag,
  Text,
} from "@synnaxlabs/pluto";
import { location, state } from "@synnaxlabs/x";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Label } from "@/platform/label";
import { View } from "@/platform/view";

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
    <Menu.Menu level="small" gap="small">
      <Label.SelectMultiple
        value={request.hasLabels ?? []}
        onChange={(labels) => handleRequestChange((r) => ({ ...r, hasLabels: labels }))}
        triggerProps={{ hideTags: true, variant: "text" }}
        location={{ targetCorner: location.TOP_RIGHT, dialogCorner: location.TOP_LEFT }}
      />
    </Menu.Menu>
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
      className={CSS.B("range-select-filters-dialog")}
      borderColor={6}
      pack={false}
    >
      <FilterContextMenu request={request} onRequestChange={onRequestChange} />
    </Dialog.Dialog>
  </Dialog.Frame>
);

interface HasLabelsFilterProps {
  request: ranger.RetrieveRequest;
}

interface TagsProps {
  keys: label.Key[];
}

const Tags = ({ keys }: TagsProps) => {
  const labels = PLabel.useRetrieveMultiple({ keys });
  return labels.map(({ color, key, name }) => (
    <Tag.Tag key={key} color={color} size="small" textColor={9}>
      {name}
    </Tag.Tag>
  ));
};

const HasLabelsFilter = ({ request: { hasLabels } }: HasLabelsFilterProps) => {
  if (hasLabels == null || hasLabels.length === 0) return null;
  return (
    <Flex.Box x pack background={0}>
      <View.FilterChip el="span" background={0} color={9}>
        <Icon.Label />
        Labels
      </View.FilterChip>
      <Errors.SuspenseBoundary loading={null}>
        <Tags keys={hasLabels} />
      </Errors.SuspenseBoundary>
    </Flex.Box>
  );
};

export const Filters = ({ request }: SelectFiltersProps) => (
  <Flex.Box x>
    <HasLabelsFilter request={request} />
  </Flex.Box>
);
