// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

/**
 * @returns the panel's icon in its color. An absent or unknown icon key draws the
 * default panel icon, since a newer Console may store a key this one lacks.
 */
export const renderIcon = ({
  icon,
  color,
}: Pick<panel.Panel, "icon" | "color">): Icon.ReactElement => {
  const Glyph = (icon == null ? undefined : Icon.getPickable(icon)?.icon) ?? Icon.Panel;
  return <Glyph {...Icon.tint(color)} />;
};
