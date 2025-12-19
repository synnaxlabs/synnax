// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError } from "@synnaxlabs/client";
import { Channel, Flex, Form, Text, Tooltip } from "@synnaxlabs/pluto";
import { location, type optional, primitive, status } from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo } from "react";

import { CSS } from "@/css";
import { useSelectActiveKey as useSelectActiveRangeKey } from "@/range/selectors";

export interface ChannelNameProps
  extends optional.Optional<Omit<Text.MaybeEditableProps, "value">, "level"> {
  channel: channel.Key;
  defaultName?: string;
  namePath: string;
}

export const ChannelName = ({
  channel,
  defaultName = "No Channel",
  className,
  namePath,
  ...rest
}: ChannelNameProps) => {
  const { onChange } = Form.useField<string>(namePath);
  const formName = Form.useFieldValue<string>(namePath);
  const range = useSelectActiveRangeKey();
  const { data, retrieve, ...restResult } = Channel.useRetrieveStateful();
  useEffect(() => {
    if (primitive.isZero(channel)) return;
    retrieve({ key: channel, rangeKey: range ?? undefined });
  }, [channel, range]);
  const { update } = Channel.useRename();
  const name = data?.name ?? (primitive.isNonZero(formName) ? formName : defaultName);
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
    if (primitive.isZero(channel))
      return { variant: "warning", message: "No channel selected" };
    if (
      restResult.status.variant === "error" &&
      NotFoundError.matches(restResult.status.details.error) &&
      restResult.status.details.error.message.includes("Channel")
    )
      return {
        variant: "error",
        message: "Channel not found. Was it deleted?",
        description:
          "If it was deleted, a new channel will be created when the task is configured.",
      };
    return restResult.status;
  }, [restResult.status, channel, range]);

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
      <Flex.Box y gap="small">
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
