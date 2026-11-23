// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import type { Icon } from "@synnaxlabs/lyra/icon";
import type { ReactElement } from "react";

import { Calculation } from "@/calculation";
import { resolveIcon } from "@/channel/resolveIcon";

export interface KindIconProps extends Icon.IconProps {
  channel: Pick<channel.Payload, "key" | "dataType">;
}

/** The icon of a channel. It shows a calculation icon when a calculation writes it. */
export const KindIcon = ({ channel: ch, ...rest }: KindIconProps): ReactElement => {
  const Resolved = resolveIcon(ch, Calculation.useIsOutput(ch.key));
  return <Resolved {...rest} />;
};
