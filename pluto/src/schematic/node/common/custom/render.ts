// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x/deep";
import { dimensions } from "@synnaxlabs/x/dimensions";
import { direction } from "@synnaxlabs/x/direction";
import { type location } from "@synnaxlabs/x/location";
import { useRef } from "react";

const ORIGINAL_STROKE_ATTRIBUTE = "data-original-stroke";
const ORIGINAL_FILL_ATTRIBUTE = "data-original-fill";

const iterElements = (
  state: schematic.symbol.State,
  svgElement: Element,
  fn: (el: Element, region: schematic.symbol.Region) => void,
) => {
  state.regions.forEach((region) => {
    region.selectors.forEach((selector) => {
      const elements = svgElement.querySelectorAll(selector);
      elements.forEach((el) => fn(el, region));
    });
  });
};

const applyOriginalAttributes = (el: Element) => {
  const prevStroke = el.getAttribute(ORIGINAL_STROKE_ATTRIBUTE);
  const prevFill = el.getAttribute(ORIGINAL_FILL_ATTRIBUTE);
  if (prevStroke != null) el.setAttribute("stroke", prevStroke);
  if (prevFill != null) el.setAttribute("fill", prevFill);
};

const storeOriginalAttributes = (el: Element) => {
  if (!el.hasAttribute(ORIGINAL_STROKE_ATTRIBUTE)) {
    const originalStroke = el.getAttribute("stroke");
    if (originalStroke != null)
      el.setAttribute(ORIGINAL_STROKE_ATTRIBUTE, originalStroke);
  }
  if (!el.hasAttribute(ORIGINAL_FILL_ATTRIBUTE)) {
    const originalFill = el.getAttribute("fill");
    if (originalFill != null) el.setAttribute(ORIGINAL_FILL_ATTRIBUTE, originalFill);
  }
};

const applyState = (
  svgElement: Element,
  state: schematic.symbol.State,
  prevState?: schematic.symbol.State | null,
) => {
  if (prevState != null) iterElements(prevState, svgElement, applyOriginalAttributes);
  iterElements(state, svgElement, (el, region) => {
    storeOriginalAttributes(el);

    const { strokeColor, fillColor } = region;

    if (strokeColor != null) el.setAttribute("stroke", strokeColor);
    else {
      const originalStroke = el.getAttribute(ORIGINAL_STROKE_ATTRIBUTE);
      if (originalStroke != null) el.setAttribute("stroke", originalStroke);
    }

    if (fillColor != null) el.setAttribute("fill", fillColor);
    else {
      const originalFill = el.getAttribute(ORIGINAL_FILL_ATTRIBUTE);
      if (originalFill != null) el.setAttribute("fill", originalFill);
    }
  });
};

export interface UseRenderArgs {
  container: HTMLElement | null;
  orientation: location.Outer;
  activeState: string;
  externalScale: number;
  spec?: schematic.symbol.Spec;
  onMount?: (svgElement: SVGSVGElement) => void;
  stateOverrides?: schematic.symbol.State[];
}

export const useRender = ({
  container,
  orientation,
  activeState,
  externalScale,
  spec,
  onMount,
  stateOverrides,
}: UseRenderArgs) => {
  const svgElementRef = useRef<SVGSVGElement>(null);
  const baseDimsRef = useRef<dimensions.Dimensions>({ width: 0, height: 0 });

  const prevExternalScaleRef = useRef<number | undefined>(undefined);
  const prevOrientationRef = useRef<location.Outer | undefined>(undefined);
  const prevSpecDataRef = useRef<schematic.symbol.Spec | undefined>(undefined);
  const prevStateRef = useRef<schematic.symbol.State>(undefined);
  const prevStateOverridesRef = useRef<typeof stateOverrides>(undefined);

  if (spec == null || spec.svg.length === 0 || container == null) return;

  const externalScaleDiffers = prevExternalScaleRef.current !== externalScale;
  const svgDiffers = prevSpecDataRef.current?.svg !== spec?.svg;
  const orientationDiffers = prevOrientationRef.current !== orientation;
  const internalScaleDiffers = prevSpecDataRef.current?.scale !== spec?.scale;
  const scaleStrokeDiffers = prevSpecDataRef.current?.scaleStroke !== spec?.scaleStroke;
  const specDiffers = prevSpecDataRef.current !== spec;

  const stateIndex = activeState === "active" ? 1 : 0;
  const currState = stateOverrides?.[stateIndex] ?? spec.states[stateIndex];

  const stateDiffers = prevStateRef.current !== currState;
  const stateOverridesDiffers =
    JSON.stringify(prevStateOverridesRef.current) !== JSON.stringify(stateOverrides);
  const different =
    externalScaleDiffers ||
    svgDiffers ||
    scaleStrokeDiffers ||
    stateDiffers ||
    stateOverridesDiffers;
  if (!different) return;
  if (externalScaleDiffers) prevExternalScaleRef.current = externalScale;
  if (orientationDiffers) prevOrientationRef.current = orientation;
  if (specDiffers) prevSpecDataRef.current = deep.copy(spec);
  if (stateOverridesDiffers) prevStateOverridesRef.current = stateOverrides;
  const { svg, scaleStroke, scale } = spec;
  if (svgElementRef.current == null || svgDiffers) {
    if (svgElementRef.current != null) {
      svgElementRef.current.remove();
      svgElementRef.current = null;
    }
    const parser = new DOMParser();
    const doc = parser.parseFromString(svg, "image/svg+xml");
    const svgElement = doc.documentElement;
    svgElementRef.current = svgElement as unknown as SVGSVGElement;

    const viewBoxAttr = svgElementRef.current.getAttribute("viewBox");
    if (viewBoxAttr) {
      const [, , width, height] = viewBoxAttr.split(" ").map(Number);
      baseDimsRef.current = { width, height };
    } else if (svgElementRef.current.viewBox?.baseVal)
      baseDimsRef.current = {
        width: svgElementRef.current.viewBox.baseVal.width,
        height: svgElementRef.current.viewBox.baseVal.height,
      };
    else baseDimsRef.current = { width: 100, height: 100 };

    const existingG = svgElement.querySelector("g");
    if (!existingG) {
      const gElement = doc.createElementNS("http://www.w3.org/2000/svg", "g");
      const children = Array.from(svgElement.children);
      children.forEach((child) => svgElement.removeChild(child));
      children.forEach((child) => {
        if (child !== gElement) gElement.appendChild(child);
      });
      svgElement.appendChild(gElement);
    }
    container.appendChild(svgElement);
    onMount?.(svgElementRef.current);
  }

  if (stateDiffers || stateOverridesDiffers) {
    applyState(svgElementRef.current, currState, prevStateRef.current);
    prevStateRef.current = deep.copy(currState);
  }

  if (
    internalScaleDiffers ||
    externalScaleDiffers ||
    orientationDiffers ||
    svgDiffers
  ) {
    let preScaledDims = baseDimsRef.current;
    if (direction.construct(orientation) === "y")
      preScaledDims = dimensions.swap(preScaledDims);
    const scaledDims = dimensions.scale(preScaledDims, scale * externalScale);
    svgElementRef.current.setAttribute("width", scaledDims.width.toString());
    svgElementRef.current.setAttribute("height", scaledDims.height.toString());
    svgElementRef.current.setAttribute(
      "viewBox",
      `0 0 ${preScaledDims.width} ${preScaledDims.height}`,
    );
  }

  if (scaleStrokeDiffers) {
    const pathElements = svgElementRef.current.querySelectorAll(
      "path, circle, rect, line, ellipse, polygon, polyline",
    );
    if (!scaleStroke)
      pathElements.forEach((el) =>
        el.setAttribute("vector-effect", "non-scaling-stroke"),
      );
    else pathElements.forEach((el) => el.removeAttribute("vector-effect"));
  }
};
