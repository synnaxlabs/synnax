import { type ranger } from "@synnaxlabs/client";
import {
  Dialog,
  Flex,
  Icon,
  Menu as PMenu,
  Text,
  type state,
  Label as PLabel,
  Tag,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { Label } from "@/label";
export interface SelectFiltersProps {
  request: ranger.RetrieveRequest;
  onRequestChange: state.Setter<ranger.RetrieveRequest>;
}

export const FilterContextMenu = ({ request, onRequestChange }: SelectFiltersProps) => (
  <PMenu.Menu level="small" gap="small">
    <Label.SelectMultiple
      value={request.hasLabels ?? []}
      onChange={(v) => onRequestChange((p) => ({ ...p, hasLabels: v }))}
      triggerProps={{ hideTags: true, variant: "text" }}
      location={{ targetCorner: location.TOP_RIGHT, dialogCorner: location.TOP_LEFT }}
    />
  </PMenu.Menu>
);

export const SelectFilters = ({ request, onRequestChange }: SelectFiltersProps) => (
  <Dialog.Frame location={location.BOTTOM_LEFT}>
    <Dialog.Trigger hideCaret>
      <Icon.Filter />
    </Dialog.Trigger>
    <Dialog.Dialog
      background={1}
      style={{
        padding: "1rem",
      }}
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
  if (request.hasLabels == null || request.hasLabels.length === 0) return null;
  const labels = PLabel.retrieveMultiple.useDirect({
    params: { keys: request.hasLabels },
  });
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
      {labels.data?.map((l) => (
        <Tag.Tag key={l.key} color={l.color} size="small" textColor={9}>
          {l.name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};

export const Filters = ({ request }: SelectFiltersProps) => {
  return (
    <Flex.Box x>
      <HasLabelsFilter request={request} />
    </Flex.Box>
  );
};
