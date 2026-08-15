// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/base/Orbital.css";

import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

interface RingProps {
  variant: "a" | "b" | "c";
}

const Ring = ({ variant }: RingProps): ReactElement => (
  <div className={CSS(CSS.BE("orbital", "tilt"), CSS.BEM("orbital", "tilt", variant))}>
    <div className={CSS.BE("orbital", "ring")}>
      {variant !== "c" && <div className={CSS.BE("orbital", "laser")} />}
    </div>
  </div>
);

export interface OrbitalProps extends Flex.BoxProps {
  /**
   * Occupies the orbital's center in place of the default sphere. Give it an
   * opaque background so ring arcs passing behind are occluded.
   */
  core?: ReactNode;
}

/**
 * Orbital is the large loading figure: two laser rings tumbling on crossed axes
 * plus a slow dashed ring on a third, around a central core. Sized by
 * `--pluto-orbital-size`. It carries no appearance delay of its own; wrap it in
 * {@link Loading} for a surface that may resolve quickly.
 */
export const Orbital = ({ core, className, ...rest }: OrbitalProps): ReactElement => (
  <Flex.Box center full className={CSS(CSS.B("orbital"), className)} {...rest}>
    <div className={CSS.BE("orbital", "stage")}>
      <div
        className={CSS(
          CSS.BE("orbital", "core"),
          core == null && CSS.BEM("orbital", "core", "orb"),
        )}
      >
        {core}
      </div>
      <Ring variant="a" />
      <Ring variant="b" />
      <Ring variant="c" />
    </div>
  </Flex.Box>
);
