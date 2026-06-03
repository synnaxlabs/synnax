// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/offPageReference/offPageReference.css";

import { color, direction } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { CSS } from "@/css";
import { resolveColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/offPageReference/config";
import { Text } from "@/text";
import { Theming } from "@/theming";

export const offPageReferenceTooltip = (
  page?: string,
  dblClickNav?: boolean,
): string | undefined => {
  if (page == null || page.length === 0) return undefined;
  const mode = dblClickNav !== false ? "Double" : "Single";
  return `${mode}-click to navigate`;
};

interface RenderProps extends Omit<
  Config,
  "label" | "page" | "dblClickNav" | "variant"
> {
  id?: string;
  label?: string;
  className?: string;
  title?: string;
  onLabelChange?: (label: string) => void;
}

export const OffPageReference = ({
  id,
  className,
  orientation = "right",
  label = "text",
  color: colorVal,
  level = "p",
  onLabelChange,
}: RenderProps): ReactElement => {
  const element = document.querySelector(`[data-id="${id}"]`);
  if (element) element.classList.add(orientation);

  const swap = direction.construct(orientation) === "y";
  const theme = Theming.use();
  const resolvedColor = resolveColor(colorVal, theme);
  const textColor = color.pickByContrast(
    resolvedColor,
    theme.colors.text,
    theme.colors.textInverted,
  );

  return (
    <Primitive.Div
      className={CSS(CSS.B("arrow"), CSS.loc(orientation), className)}
      orientation={orientation}
    >
      <div className="wrapper">
        <div
          className="outline"
          style={
            {
              "--off-page-color": color.cssString(resolvedColor),
              "--off-page-text-color": color.cssString(textColor),
            } as CSSProperties
          }
        >
          <div className="bg">
            <Text.MaybeEditable
              value={label}
              onChange={onLabelChange}
              level={level}
              className={CSS.BE("symbol", "label")}
            />
          </div>
        </div>
      </div>
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle
          location="left"
          orientation={orientation}
          preventAutoAdjust
          left={98}
          top={50}
          swap={swap}
          id="1"
        />
        <Handle.Handle
          location="right"
          preventAutoAdjust
          orientation={orientation}
          left={1}
          top={50}
          swap={swap}
          id="2"
        />
      </Handle.Boundary>
      <svg
        style={{ visibility: "hidden", position: "absolute" }}
        width="0"
        height="0"
        xmlns="http://www.w3.org/2000/svg"
        version="1.1"
      >
        <defs>
          <filter id="goo">
            <feGaussianBlur in="SourceGraphic" stdDeviation="1.5" result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 19 -9"
              result="goo"
            />
            <feComposite in="SourceGraphic" in2="goo" operator="atop" />
          </filter>
        </defs>
      </svg>
    </Primitive.Div>
  );
};
