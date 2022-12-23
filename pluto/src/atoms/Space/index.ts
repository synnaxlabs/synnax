import {
  Space as CoreSpace,
  SpaceAlignment,
  SpaceAlignments,
  SpaceJustification,
  SpaceJustifications,
} from "./Space";
import { SpaceCentered } from "./SpaceCentered";
export type { SpaceProps } from "./Space";
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
