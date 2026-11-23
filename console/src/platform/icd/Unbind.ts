// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Channel } from "@synnaxlabs/pluto";
import type { deep } from "@synnaxlabs/x";
import { useEffect } from "react";

export interface UnbindProps {
  path: deep.Target<channel.Key>;
  channel: channel.Key;
  /** Whether the field of the binding takes a channel of the data type. */
  takes: (dataType: string) => boolean;
}

/** Unbinds a channel that its field no longer takes, so deploying creates a new one. */
export const Unbind = ({ path, channel: key, takes }: UnbindProps): null => {
  const { set } = PForm.useContext();
  const dataType = Channel.useResultDataType({ key }).data?.toString();
  const taken = dataType == null || takes(dataType);
  useEffect(() => {
    if (!taken) set(path, 0);
  }, [taken]);
  return null;
};
