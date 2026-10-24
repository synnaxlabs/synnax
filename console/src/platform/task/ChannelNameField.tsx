// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Access, Channel } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useIsPreview } from "@/platform/task/Form";

export interface ChannelNameFieldProps {
  /** The channel the name belongs to, or 0 before configure creates it. */
  channel: channel.Key;
  /** Holds the name configure gives the channel it creates. */
  namePath: string;
  /** The name configure falls back to; shown as the placeholder. */
  defaultName?: string;
}

/**
 * A labeled input that names a channel. Until configure creates the channel, the name
 * lives in the form at namePath; after, an edit renames the channel itself.
 */
export const ChannelNameField = ({
  channel: key,
  namePath,
  defaultName = "",
}: ChannelNameFieldProps): ReactElement =>
  key === 0 ? (
    <Form.TextField
      path={namePath}
      label="Channel"
      padHelpText={false}
      inputProps={{ placeholder: defaultName === "" ? "Channel name" : defaultName }}
    />
  ) : (
    <Existing channel={key} />
  );

const Existing = ({ channel: key }: { channel: channel.Key }) => {
  const { data: name = "" } = Channel.useResultName({ key });
  const { update } = Channel.useRename();
  const canRename = Access.useUpdateGranted(channel.TYPE_ONTOLOGY_ID);
  const isPreview = useIsPreview();
  const handleChange = useCallback(
    (next: string) => {
      if (next.length > 0 && next !== name) update({ key, name: next });
    },
    [key, name, update],
  );
  return (
    <Input.Item label="Channel" padHelpText={false}>
      <Input.Text
        value={name}
        onChange={handleChange}
        onlyChangeOnBlur
        disabled={isPreview || !canRename}
      />
    </Input.Item>
  );
};
