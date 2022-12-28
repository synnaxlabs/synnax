import type { ChannelPayload } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/pluto";
import type { ListColumn, SelectMultipleProps, SelectProps } from "@synnaxlabs/pluto";

const channelColumns: Array<ListColumn<ChannelPayload>> = [
  {
    key: "name",
    label: "Name",
  },
];

const verboseChannelColumns: Array<ListColumn<ChannelPayload>> = [
  {
    key: "name",
    label: "Name",
  },
  {
    key: "rate",
    label: "Rate",
  },
  {
    key: "dataType",
    label: "Data Type",
  },
];

export interface SelectMultipleChannelsProps
  extends Omit<SelectMultipleProps<ChannelPayload>, "columns"> {
  verbose?: boolean;
}

export const SelectMultipleChannels = ({
  verbose = false,
  ...props
}: SelectMultipleChannelsProps): JSX.Element => (
  <Select.Multiple
    columns={verbose ? verboseChannelColumns : channelColumns}
    {...props}
  />
);

export interface SelectChannelProps
  extends Omit<SelectProps<ChannelPayload>, "columns"> {
  verbose?: boolean;
}

export const SelectChanel = ({
  verbose = false,
  ...props
}: SelectChannelProps): JSX.Element => (
  <Select columns={verbose ? verboseChannelColumns : channelColumns} {...props} />
);
