// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import geistMono from "@fontsource/geist-mono/files/geist-mono-latin-400-normal.woff2";
import inter200 from "@fontsource/inter/files/inter-latin-200-normal.woff2";
import inter300 from "@fontsource/inter/files/inter-latin-300-normal.woff2";
import inter400 from "@fontsource/inter/files/inter-latin-400-normal.woff2";
import inter500 from "@fontsource/inter/files/inter-latin-500-normal.woff2";
import inter600 from "@fontsource/inter/files/inter-latin-600-normal.woff2";
import inter700 from "@fontsource/inter/files/inter-latin-700-normal.woff2";
import inter800 from "@fontsource/inter/files/inter-latin-800-normal.woff2";
import inter900 from "@fontsource-variable/inter/files/inter-latin-standard-normal.woff2";
import { Theming as Base } from "@synnaxlabs/lyra/theming";
import { type PropsWithChildren, type ReactElement, useEffect } from "react";

import { Aether } from "@/aether";
import { theming } from "@/theming/aether";

const FONT_URLS = [
  { name: "Inter Nine", url: inter900 },
  { name: "Inter Eight", url: inter800 },
  { name: "Inter Seven", url: inter700 },
  { name: "Inter Six", url: inter600 },
  { name: "Inter Five", url: inter500 },
  { name: "Inter Four", url: inter400 },
  { name: "Inter Three", url: inter300 },
  { name: "Inter Two", url: inter200 },
  { name: "Geist Mono", url: geistMono },
];

const WorkerBridge = ({ children }: PropsWithChildren): ReactElement => {
  const theme = Base.use();
  const [{ path }, , setState] = Aether.use({
    type: theming.Provider.TYPE,
    schema: theming.Provider.z,
    initialState: { theme, fontURLs: FONT_URLS },
  });
  // The theme.key dep will not trigger a re-render if the theme properties change
  // but not the key. This reduces re-renders, but should be corrected once the user
  // has the ability to edit the theme themselves.
  useEffect(() => setState((p) => ({ ...p, theme })), [theme.key]);
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};

/** Props for {@link Provider}. */
export interface ProviderProps extends Base.ProviderProps {}

/**
 * The lyra theming provider plus a bridge that hands the theme and its fonts to the
 * aether worker thread. Mount it inside the Aether provider.
 */
export const Provider = ({ children, ...rest }: ProviderProps): ReactElement => (
  <Base.Provider {...rest}>
    <WorkerBridge>{children}</WorkerBridge>
  </Base.Provider>
);
