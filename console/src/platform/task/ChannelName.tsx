// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError, query, status } from "@synnaxlabs/client";
import { Channel, Flex, Form, Synnax, Text, Tooltip } from "@synnaxlabs/pluto";
import { errors, location, type optional, primitive } from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo, useState } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

export interface ChannelNameProps extends optional.Optional<
  Omit<Text.MaybeEditableProps, "value">,
  "level"
> {
  channel: channel.Key;
  defaultName?: string;
  namePath: string;
}

export const ChannelName = ({
  channel,
  defaultName = "No channel",
  className,
  namePath,
  ...rest
}: ChannelNameProps) => {
  // Use optional values here because for some reason, the namePath is not always
  // populated. Eventually we should strongly type the Form so we don't need to worry
  // about this.
  const fieldCtx = Form.useField<string>(namePath, { optional: true });
  const onChange = fieldCtx?.onChange;
  const formName = Form.useFieldValue<string>(namePath, { optional: true });
  const range = Session.Range.useSelectSelectedKey();
  const client = Synnax.use();
  const [data, setData] = useState<channel.Channel | undefined>(undefined);
  const [error, setError] = useState<Error | null>(null);
  useEffect(() => {
    setData(undefined);
    setError(null);
    if (channel === 0 || client == null) return;
    let cancelled = false;
    const fetchChannel = async () => {
      try {
        const ch = await client.channels.retrieve({
          key: channel,
          rangeKey: range ?? undefined,
        });
        if (!cancelled) setData(ch);
      } catch (e) {
        if (!cancelled) setError(errors.fromUnknown(e));
      }
    };
    void fetchChannel();
    const disconnect = client.channels.onChange(
      { key: channel, rangeKey: range ?? undefined },
      (res) => {
        if (query.isLive(res)) setData(res);
      },
    );
    return () => {
      cancelled = true;
      disconnect();
    };
  }, [client, channel, range]);
  const { update } = Channel.useRename();
  const name = getName(data, formName, defaultName);
  const handleRename = useCallback(
    (name: string) => {
      if (channel === 0) {
        onChange?.(name);
        return;
      }
      update({ key: channel, name });
    },
    [channel, update, onChange],
  );
  const stat = useMemo((): Pick<
    status.Status,
    "variant" | "message" | "description"
  > => {
    if (channel === 0)
      return { variant: "warning", message: "No channel selected", description: "" };
    if (error != null) {
      if (NotFoundError.matches(error) && error.message.includes("Channel"))
        return {
          variant: "error",
          message: "Channel not found. Was it deleted?",
          description:
            "If it was deleted, a new channel will be created when the task is configured.",
        };
      return {
        variant: "error",
        message: "Failed to retrieve channel",
        description: error.message,
      };
    }
    if (data == null)
      return { variant: "loading", message: "Retrieving channel", description: "" };
    return {
      variant: "success",
      message: "Successfully retrieved channel",
      description: "",
    };
  }, [error, data, channel]);

  const variant = status.removeVariants(stat.variant, ["success"]);
  const text = (
    <Text.MaybeEditable
      className={CSS(className, CSS.BE("task", "channel-name"))}
      status={variant}
      level="small"
      value={name}
      onChange={handleRename}
      allowDoubleClick={false}
      overflow="ellipsis"
      {...rest}
    />
  );

  if (stat.variant !== "success" && stat.variant !== "error" && variant !== "warning")
    return text;
  return (
    <Tooltip.Dialog location={location.CENTER_RIGHT}>
      <Flex.Box y gap="small" className={CSS.B("task-channel-name")}>
        <Text.Text status={variant} level="p" color={10} weight={500}>
          {stat.message}
        </Text.Text>
        {stat.description != null && (
          <Text.Text level="small" color={9} weight={450}>
            {stat.description}
          </Text.Text>
        )}
      </Flex.Box>
      {text}
    </Tooltip.Dialog>
  );
};

const getName = (
  data: channel.Channel | undefined,
  formName: string | null,
  defaultName: string,
) => {
  if (data != null) {
    if (primitive.isNonZero(data.alias)) return data.alias;
    return data.name;
  }
  if (primitive.isNonZero(formName)) return formName;
  return defaultName;
};
