// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type ReactElement, useState } from "react";

import { Button } from "@/button";
import { Channel } from "@/channel";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Status } from "@/status/core";
import { Text } from "@/text";

export interface AliasInputProps extends Input.TextProps {
  channel: channel.Key;
  range?: string;
  shadow?: boolean;
}

export const AliasInput = ({
  channel,
  range,
  shadow,
  className,
  ...rest
}: AliasInputProps): ReactElement => {
  const { value, onChange } = rest;
  const [loading, setLoading] = useState(false);
  const { update } = Channel.useUpdateAlias();
  const { data } = Channel.useRetrieve({ key: channel, rangeKey: range });
  const setAlias = async (value: string) => {
    update({ alias: value, range, channel });
  };
  const alias = data?.alias;
  const name = data?.alias ?? data?.name;
  let icon = <Icon.Rename />;
  if (loading) icon = <Icon.Loading />;
  else if (alias === value) icon = <Icon.Check />;
  const canSetAlias = setAlias != null && !loading && alias !== value && channel !== 0;
  const handleError = Status.useErrorHandler();
  const handleSetAlias = (): void => {
    if (!canSetAlias) return;
    handleError(async () => {
      setLoading(true);
      try {
        await setAlias(value);
        setLoading(false);
      } catch (e) {
        setLoading(false);
        throw e;
      }
    }, "Failed to set channel alias");
  };

  const handleSetValueToAlias = (): void => {
    if (alias == null) return;
    onChange?.(alias);
  };

  const setAliasTooltip =
    channel === 0 ? (
      <Text.Text level="small">
        Select a channel to enable alias syncing with this label
      </Text.Text>
    ) : setAlias == null ? (
      <Text.Text level="small">
        Select a range to enable alias syncing with this label
      </Text.Text>
    ) : value.length === 0 ? (
      <Text.Text level="small">
        Enter a value to enable alias syncing with this label
      </Text.Text>
    ) : alias === value ? (
      <Text.Text level="small">Alias synced with this label</Text.Text>
    ) : (
      <Text.Text level="small">Sync alias for {name} with this label</Text.Text>
    );

  return (
    <Input.Text selectOnFocus {...rest}>
      {canSetAlias && (
        <Button.Button
          onClick={handleSetValueToAlias}
          tooltip={<Text.Text level="small">Set {name} as label</Text.Text>}
          tooltipLocation={{ y: "top" }}
          variant="outlined"
        >
          <Icon.Sync />
        </Button.Button>
      )}
      <Button.Button
        onClick={handleSetAlias}
        disabled={!canSetAlias}
        tooltip={setAliasTooltip}
        tooltipLocation={{ y: "top" }}
        variant="outlined"
      >
        {icon}
      </Button.Button>
    </Input.Text>
  );
};
