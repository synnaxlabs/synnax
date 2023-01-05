// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Space as CoreSpace,
  SpaceAlignment,
  SpaceAlignments,
  SpaceJustification,
  SpaceJustifications,
} from "./Space";
import { SpaceCentered } from "./SpaceCentered";
export type { SpaceProps, SpaceExtensionProps } from "./Space";
export type { SpaceCenteredProps } from "./SpaceCentered";
export type { SpaceAlignment, SpaceJustification };

type CoreSpaceType = typeof CoreSpace;

interface SpaceType extends CoreSpaceType {
  Centered: typeof SpaceCentered;
  Justifications: readonly SpaceJustification[];
  Alignments: readonly SpaceAlignment[];
}

export const Space = CoreSpace as SpaceType;

Space.Centered = SpaceCentered;
Space.Justifications = SpaceJustifications;
Space.Alignments = SpaceAlignments;
