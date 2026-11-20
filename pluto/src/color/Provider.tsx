// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color as Base } from "@synnaxlabs/lyra/color";
import type { ReactElement } from "react";

import { Aether } from "@/aether";
import { color } from "@/color/aether";
import { useSeriesValues } from "@/color/library";

export interface ProviderProps extends Omit<Base.ProviderProps, "presets"> {}

/**
 * The Lyra color provider with the series colors as picker presets. It also gives the
 * series colors to the Aether components below. Mount it inside the Synnax and theming
 * providers.
 */
export const Provider = ({ children, ...rest }: ProviderProps): ReactElement => {
  const colors = useSeriesValues();
  const { path } = Aether.useUnidirectional({
    type: color.SeriesProvider.TYPE,
    schema: color.SeriesProvider.z,
    state: { colors },
  });
  return (
    <Base.Provider {...rest} presets={colors}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Base.Provider>
  );
};
