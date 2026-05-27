// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { deep, dimensions, direction, type location } from "@synnaxlabs/x";
import { type RefCallback, useCallback, useRef } from "react";

import { useInitializerRef, useSyncedRef } from "@/hooks/ref";

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
  orientation: location.Outer;
  activeState: string;
  externalScale: number;
  spec?: schematic.symbol.Spec;
  onMount?: (svgElement: SVGSVGElement) => void;
  stateOverrides?: schematic.symbol.State[];
}

interface RenderState {
  svgElement: SVGSVGElement | null;
  baseDims: dimensions.Dimensions;
  prevExternalScale: number | undefined;
  prevOrientation: location.Outer | undefined;
  prevSpecData: schematic.symbol.Spec | undefined;
  prevState: schematic.symbol.State | undefined;
  prevStateOverrides: schematic.symbol.State[] | undefined;
}

const createRenderState = (): RenderState => ({
  svgElement: null,
  baseDims: { width: 0, height: 0 },
  prevExternalScale: undefined,
  prevOrientation: undefined,
  prevSpecData: undefined,
  prevState: undefined,
  prevStateOverrides: undefined,
});

const runRender = (container: HTMLElement, args: UseRenderArgs, state: RenderState) => {
  const { orientation, activeState, externalScale, spec, onMount, stateOverrides } =
    args;
  if (spec == null || spec.svg.length === 0) return;

  const externalScaleDiffers = state.prevExternalScale !== externalScale;
  const svgDiffers = state.prevSpecData?.svg !== spec?.svg;
  const orientationDiffers = state.prevOrientation !== orientation;
  const internalScaleDiffers = state.prevSpecData?.scale !== spec?.scale;
  const scaleStrokeDiffers = state.prevSpecData?.scaleStroke !== spec?.scaleStroke;
  const specDiffers = state.prevSpecData !== spec;

  const stateIndex = activeState === "active" ? 1 : 0;
  const currState = stateOverrides?.[stateIndex] ?? spec.states[stateIndex];

  const stateDiffers = state.prevState !== currState;
  const stateOverridesDiffers = !deep.equal(state.prevStateOverrides, stateOverrides);
  const different =
    externalScaleDiffers ||
    svgDiffers ||
    scaleStrokeDiffers ||
    stateDiffers ||
    stateOverridesDiffers;
  if (!different) return;
  if (externalScaleDiffers) state.prevExternalScale = externalScale;
  if (orientationDiffers) state.prevOrientation = orientation;
  if (specDiffers) state.prevSpecData = spec;
  if (stateOverridesDiffers) state.prevStateOverrides = stateOverrides;
  const { svg, scaleStroke, scale } = spec;
  if (state.svgElement == null || svgDiffers) {
    if (state.svgElement != null) {
      state.svgElement.remove();
      state.svgElement = null;
    }
    const parser = new DOMParser();
    const doc = parser.parseFromString(svg, "image/svg+xml");
    const svgElement = doc.documentElement;
    state.svgElement = svgElement as unknown as SVGSVGElement;

    const viewBoxAttr = state.svgElement.getAttribute("viewBox");
    if (viewBoxAttr) {
      const [, , width, height] = viewBoxAttr.split(" ").map(Number);
      state.baseDims = { width, height };
    } else if (state.svgElement.viewBox?.baseVal)
      state.baseDims = {
        width: state.svgElement.viewBox.baseVal.width,
        height: state.svgElement.viewBox.baseVal.height,
      };
    else state.baseDims = { width: 100, height: 100 };

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
    onMount?.(state.svgElement);
  }

  if (stateDiffers || stateOverridesDiffers) {
    applyState(state.svgElement, currState, state.prevState);
    state.prevState = currState;
  }

  if (
    internalScaleDiffers ||
    externalScaleDiffers ||
    orientationDiffers ||
    svgDiffers
  ) {
    let preScaledDims = state.baseDims;
    if (direction.construct(orientation) === "y")
      preScaledDims = dimensions.swap(preScaledDims);
    const scaledDims = dimensions.scale(preScaledDims, scale * externalScale);
    state.svgElement.setAttribute("width", scaledDims.width.toString());
    state.svgElement.setAttribute("height", scaledDims.height.toString());
    state.svgElement.setAttribute(
      "viewBox",
      `0 0 ${preScaledDims.width} ${preScaledDims.height}`,
    );
  }

  if (scaleStrokeDiffers) {
    const pathElements = state.svgElement.querySelectorAll(
      "path, circle, rect, line, ellipse, polygon, polyline",
    );
    if (!scaleStroke)
      pathElements.forEach((el) =>
        el.setAttribute("vector-effect", "non-scaling-stroke"),
      );
    else pathElements.forEach((el) => el.removeAttribute("vector-effect"));
  }
};

/// useRender returns a ref callback that drives the SVG mount/state/scale lifecycle
/// for a custom symbol. The returned callback is stable across renders. When the
/// container element attaches, the SVG is built and inserted; when it detaches, the
/// SVG is removed and internal diff state is cleared so the next attach re-creates
/// the SVG cleanly (including after a Missing→Resolved→Missing→Resolved cycle and
/// under StrictMode's simulated remount). Subsequent args changes against an
/// already-attached container are picked up via a render-phase pass.
export const useRender = (args: UseRenderArgs): RefCallback<HTMLElement> => {
  const containerRef = useRef<HTMLElement | null>(null);
  const argsRef = useSyncedRef(args);
  const stateRef = useInitializerRef<RenderState>(createRenderState);

  if (containerRef.current != null)
    runRender(containerRef.current, args, stateRef.current);

  return useCallback<RefCallback<HTMLElement>>((el) => {
    if (el == null) {
      const { svgElement } = stateRef.current;
      if (svgElement != null) svgElement.remove();
      stateRef.current = createRenderState();
      containerRef.current = null;
      return;
    }
    containerRef.current = el;
    runRender(el, argsRef.current, stateRef.current);
  }, []);
};
