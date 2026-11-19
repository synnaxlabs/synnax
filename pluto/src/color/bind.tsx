// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, group } from "@synnaxlabs/client";
import { type Color } from "@synnaxlabs/lyra/color";
import { type color as xcolor } from "@synnaxlabs/x";

import { Access } from "@/access";
import { CreatePalette, CreateSwatch } from "@/color/Create";
import { SwatchMenu } from "@/color/Edit";
import { usePalettes, useResolve } from "@/color/library";
import { Synnax } from "@/synnax";

/** The props that bind a Lyra color control to a stored color reference. */
export interface Binding {
  value?: xcolor.Color;
  onChange: (value?: xcolor.Color) => void;
  library?: Color.Library;
}

/**
 * Binds a Lyra color control to a stored color reference. A pick from a palette calls
 * onChange with a named reference, a custom color with a fixed reference, and Auto with
 * undefined. The picker can also create a named color, which the value then follows,
 * and edit a named color, for a user with the access to do so.
 */
export const useBinding = (
  value: color.Ref | undefined,
  onChange: (value?: color.Ref) => void,
): Binding => {
  const palettes = usePalettes();
  const resolve = useResolve();
  const client = Synnax.use();
  const canCreate = Access.useCreateGranted(color.TYPE_RESOURCE_ID);
  const canCreatePalette = Access.useCreateGranted(group.TYPE_RESOURCE_ID);
  return {
    value: resolve(value),
    onChange: (c) => onChange(c == null ? undefined : { type: "fixed", value: c }),
    library:
      palettes.length === 0
        ? undefined
        : {
            palettes,
            named: value?.type === "named" ? value.key : undefined,
            onNamedChange: (key) => onChange({ type: "named", key }),
            paletteEnd: canCreate ? <CreateSwatch /> : undefined,
            swatchMenu: <SwatchMenu />,
            hasSwatchMenu: (key) =>
              Access.updateGranted({ client, id: color.resourceID(key) }),
            footer: canCreate && canCreatePalette ? <CreatePalette /> : undefined,
          },
  };
};
