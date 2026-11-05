// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Channel } from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const VIDEO_QUERY: Partial<Channel.RetrieveMultipleQuery> = {
  dataTypes: [DataType.VIDEO],
};

interface SelectProps {
  value: channel.Key;
  onChange: (value: channel.Key) => void;
}

const renderSelect = Component.renderProp(({ value, onChange }: SelectProps) => (
  <Channel.SelectSingle
    value={value}
    onChange={(v: channel.Key | undefined) => onChange(v ?? 0)}
    initialQuery={VIDEO_QUERY}
    allowNone
    grow
  />
));

/** Selects the video channel at `config.channel` of a task form. */
export const VideoChannelField = (): ReactElement => (
  <PForm.Field<channel.Key>
    path="config.channel"
    label="Video channel"
    helpText="Leave empty to create a channel named after the task."
    grow
  >
    {renderSelect}
  </PForm.Field>
);
