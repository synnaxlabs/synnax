import { type ranger } from "@synnaxlabs/client";
import { Dialog, Icon, Menu as PMenu, type state } from "@synnaxlabs/pluto";
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
    />
  </PMenu.Menu>
);

export const SelectFilters = ({ request, onRequestChange }: SelectFiltersProps) => (
  <Dialog.Frame location={location.BOTTOM_LEFT}>
    <Dialog.Trigger>
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
