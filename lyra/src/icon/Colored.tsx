// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  type SVGProps,
  use,
} from "react";

import { CSS } from "@/css";
import { type FC, wrapSVGIcon } from "@/icon/Icon";
import { Theming } from "@/theming";

const Context = createContext(false);
Context.displayName = "Icon.Colored";

/**
 * Draws the third-party logos inside it in their brand colors. Use it only where the
 * logo is the main way to tell items apart. Elsewhere, logos take the text color.
 */
export const Colored = ({ children }: PropsWithChildren): ReactElement => (
  <Context value>{children}</Context>
);

/**
 * The color a brand draws its logo in on each theme's background. A string is the
 * color on both. A theme with no color keeps the text color, for brands that draw
 * their logo white on dark or black on light.
 */
export type BrandColor = string | { light?: string; dark?: string };

const useBrandColor = (c: BrandColor): string | undefined => {
  const colored = use(Context);
  const { dark } = Theming.use();
  if (!colored) return undefined;
  if (typeof c === "string") return c;
  return dark ? c.dark : c.light;
};

/** Returns the logo drawn in its brand color inside {@link Colored}. */
/* #__NO_SIDE_EFFECTS__ */
export const withBrandColor = (Logo: FC, brandColor: BrandColor): FC => {
  const O: FC = ({ style, ...rest }) => {
    const c = useBrandColor(brandColor);
    return <Logo {...rest} style={c == null ? style : { ...style, color: c }} />;
  };
  O.displayName = Logo.displayName;
  return O;
};

export interface BrandPartProps extends PropsWithChildren {
  /** The brand color of this part of the mark. */
  color: BrandColor;
}

/**
 * Draws the parts of a mark inside it in one brand color, inside {@link Colored}. Use
 * it for a mark with parts in different colors. Elsewhere, the parts take the text
 * color.
 */
export const BrandPart = ({ color, children }: BrandPartProps): ReactElement => {
  const c = useBrandColor(color);
  if (c == null) return <g>{children}</g>;
  // Icon paths take their fill from their parent, so the group holds the color.
  return (
    <g style={{ color: c }} fill="currentColor">
      {children}
    </g>
  );
};

/* #__NO_SIDE_EFFECTS__ */
export const wrapMulticolor = (
  viewBox: string,
  paths: [color: BrandColor, d: string][],
  name: string,
): FC => {
  const Multicolor = ({
    className,
    ...rest
  }: SVGProps<SVGSVGElement>): ReactElement => (
    <svg
      viewBox={viewBox}
      {...rest}
      className={CSS.cls(className, "logo")}
      fill="currentColor"
      strokeWidth={0}
      xmlns="http://www.w3.org/2000/svg"
    >
      {paths.map(([color, d]) => (
        <BrandPart key={d} color={color}>
          <path d={d} />
        </BrandPart>
      ))}
    </svg>
  );
  return wrapSVGIcon(Multicolor, name);
};
