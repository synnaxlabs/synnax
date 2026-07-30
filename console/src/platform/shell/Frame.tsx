// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/shell/Shell.css";

import { Logo } from "@synnaxlabs/media";
import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { Footer, type FooterCluster } from "@/platform/shell/Footer";
import { Nav } from "@/platform/shell/Nav";
import { Nebula } from "@/platform/shell/Nebula";

// Rotation switches shared by every pre-workspace surface. "frosted" blurs the
// nebula through the card; "inside" leaves branding to the card's content.
type CardVariant = "frosted" | "surface" | "outline" | "none";
const CARD_VARIANT: CardVariant = "frosted";

type CardShape = "sharp" | "soft" | "round";
const CARD_SHAPE: CardShape = "soft";

type LogoPlacement = "inside" | "above" | "corner";
const LOGO_PLACEMENT: LogoPlacement = "inside";

// Rounded lines around the card: "echo" = concentric fading outlines,
// "dashed" = one dashed orbit ring, "orbits" = tilted ellipses behind the card.
type Ornament = "none" | "echo" | "dashed" | "orbits";
// The cast keeps reads at the union type so rotation comparisons type-check.
const ORNAMENT = "echo" as Ornament;

const Orbits = (): ReactElement => (
  <svg
    className={CSS.BE("shell", "orbits")}
    viewBox="0 0 640 480"
    fill="none"
    stroke="currentColor"
    strokeWidth="1"
  >
    <g transform="translate(320 240)">
      <ellipse rx="300" ry="300" transform="rotate(-14) scale(1 0.44)" />
      <ellipse
        rx="240"
        ry="240"
        transform="rotate(9) scale(1 0.36)"
        strokeDasharray="3 6"
      />
      <circle
        cx="300"
        cy="0"
        r="3.5"
        fill="currentColor"
        stroke="none"
        transform="rotate(-14) scale(1 0.44) rotate(38)"
      />
      <circle
        cx="240"
        cy="0"
        r="3.5"
        fill="currentColor"
        stroke="none"
        transform="rotate(9) scale(1 0.36) rotate(-115)"
      />
    </g>
  </svg>
);

export interface FrameProps {
  /** Modifier classes for the surface; card sizing rules key off these. */
  className?: string;
  /** Target Core shown in the connection footer; omit to hide it. */
  connection?: FooterCluster | null;
  /** The card's contents. */
  children: ReactNode;
}

/**
 * Full-window shell for pre-workspace surfaces: window chrome, halftone nebula,
 * ornamented stage, and the centered card that hosts the surface's content.
 */
export const Frame = ({
  className,
  connection,
  children,
}: FrameProps): ReactElement => (
  <Flex.Box
    y
    empty
    className={CSS(
      CSS.B("shell"),
      CSS.M(`card-${CARD_VARIANT}`),
      CSS.M(`shape-${CARD_SHAPE}`),
      CSS.M(`logo-${LOGO_PLACEMENT}`),
      CSS.M(`ornament-${ORNAMENT}`),
      className,
    )}
  >
    <Nav />
    <Flex.Box
      y
      align="center"
      justify="center"
      background={1}
      gap="huge"
      grow
      data-tauri-drag-region
      className={CSS.BE("shell", "content")}
    >
      <Nebula />
      {LOGO_PLACEMENT !== "inside" && (
        <Logo
          variant="title"
          className={CSS.BE("shell", "logo")}
          data-tauri-drag-region
        />
      )}
      <Flex.Box className={CSS.BE("shell", "stage")} grow={false}>
        {ORNAMENT === "orbits" && <Orbits />}
        <Flex.Box y empty className={CSS.BE("shell", "card")} grow={false}>
          {children}
        </Flex.Box>
      </Flex.Box>
      <Footer cluster={connection} />
    </Flex.Box>
  </Flex.Box>
);
