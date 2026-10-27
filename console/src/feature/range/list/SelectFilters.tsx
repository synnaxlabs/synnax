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
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import { Label as PLabel } from "@synnaxlabs/pluto";
import { location, state } from "@synnaxlabs/x";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Label } from "@/platform/label";
import { View } from "@/platform/view";

export interface SelectFiltersProps {
  request: ranger.RetrieveMultipleParams;
  onRequestChange: state.Setter<ranger.RetrieveMultipleParams>;
}

const FilterContextMenu = ({ request, onRequestChange }: SelectFiltersProps) => {
  const handleRequestChange = (setter: state.SetArg<ranger.RetrieveMultipleParams>) => {
    onRequestChange((prev) => {
      const next = state.executeSetter(setter, prev);
      return { ...next, offset: 0, limit: undefined };
    });
  };

  return (
    <Menu.Menu level="small" gap="small">
      <Label.SelectMultiple
        value={request.where?.labels ?? []}
        onChange={(labels) =>
          handleRequestChange((r) => ({
            ...r,
            where: { ...r.where, labels: labels.length > 0 ? labels : undefined },
          }))
        }
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

interface LabelsFilterProps {
  request: ranger.RetrieveMultipleParams;
}

interface TagsProps {
  keys: label.Key[];
}

const Tags = ({ keys }: TagsProps) => {
  const labels = PLabel.useMultiple({ keys });
  return labels.map(({ color, key, name }) => (
    <Tag.Tag key={key} color={color} size="small" textColor={9}>
      {name}
    </Tag.Tag>
  ));
};

const LabelsFilter = ({ request: { where } }: LabelsFilterProps) => {
  const labels = where?.labels;
  if (labels == null || labels.length === 0) return null;
  return (
    <Flex.Box x pack background={0}>
      <View.FilterChip el="span" background={0} color={9}>
        <Icon.Label />
        Labels
      </View.FilterChip>
      <Errors.SuspenseBoundary loading={null}>
        <Tags keys={labels} />
      </Errors.SuspenseBoundary>
    </Flex.Box>
  );
};

export const Filters = ({ request }: SelectFiltersProps) => (
  <Flex.Box x>
    <LabelsFilter request={request} />
  </Flex.Box>
);
