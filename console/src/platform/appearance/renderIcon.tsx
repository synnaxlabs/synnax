// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type color } from "@synnaxlabs/x";

/** Arguments for {@link renderIcon}. */
export interface RenderIconArgs {
  icon?: string;
  color?: color.Crude;
  /** The icon drawn for an absent or unknown icon key. */
  fallback: Icon.FC;
}

/**
 * @returns the icon in its color. An absent or unknown icon key draws the fallback,
 * since a newer Console may store a key this one lacks.
 */
export const renderIcon = ({
  icon,
  color,
  fallback,
}: RenderIconArgs): Icon.ReactElement => {
  const Glyph = (icon == null ? undefined : Icon.getPickable(icon)?.icon) ?? fallback;
  return <Glyph {...Icon.tint(color)} />;
};
