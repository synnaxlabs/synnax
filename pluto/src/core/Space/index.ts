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
export type { SpaceProps, SpaceExtensionProps, SpaceElementType } from "./Space";
export type { SpaceCenteredProps } from "./SpaceCentered";
export type { SpaceAlignment, SpaceJustification };

type CoreSpaceType = typeof CoreSpace;

interface SpaceType extends CoreSpaceType {
  /**
   * A Space whose width and height is 100% and whose alignment and justification
   * is centered. Props are the same as {@link Space}.
   */
  Centered: typeof SpaceCentered;
  /** The available justifications for the Space component. */
  Justifications: readonly SpaceJustification[];
  /** The available alignments for the Space component. */
  Alignments: readonly SpaceAlignment[];
}

/**
 * A component that orients its children in a row or column and adds
 * space between them. This is essentially a thin wrapped around a
 * flex component that makes it more 'reacty' to use.
 *
 * @param props - The props for the component. All unlisted props will be passed
 * to the underlying root element.
 * @param props.align - The off axis alignment of the children. The 'off' axis is the
 * opposite direction of props.direction. For example, if direction is 'x', then the
 * off axis is 'y'. See the {@link SpaceAlignment} for available options.
 * @param props.justify - The main axis justification of the children. The 'main' axis
 * is the same direction as props.direction. For example, if direction is 'x', then the
 * main axis is 'x'. See the {@link SpaceJustification} for available options.
 * @param props.grow - A boolean or number value that determines if the space should
 * grow in the flex-box sense. A value of true will set css flex-grow to 1. A value of
 * false will leave the css flex-grow unset. A number value will set the css flex-grow
 * to that number.
 * @param props.size - A string or number value that determines the amount of spacing
 * between items. If set to "small", "medium", or "large", the spacing will be determined
 * by the theme. If set to a number, the spacing will be that number of rem.
 * @param props.wrap - A boolean value that determines if the space should wrap its
 * children.
 * @param props.el - The element type to render as. Defaults to 'div'.
 */
export const Space = CoreSpace as SpaceType;

Space.Centered = SpaceCentered;
Space.Justifications = SpaceJustifications;
Space.Alignments = SpaceAlignments;
