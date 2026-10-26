// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { useInitializerRef, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { color, dimensions, direction, type location } from "@synnaxlabs/x";
import { type RefCallback, useLayoutEffect, useRef } from "react";

import { Primitive } from "@/schematic/node/common/primitive";

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

    if (strokeColor != null) el.setAttribute("stroke", color.hex(strokeColor));
    else {
      const originalStroke = el.getAttribute(ORIGINAL_STROKE_ATTRIBUTE);
      if (originalStroke != null) el.setAttribute("stroke", originalStroke);
    }

    if (fillColor != null) el.setAttribute("fill", color.hex(fillColor));
    else {
      const originalFill = el.getAttribute(ORIGINAL_FILL_ATTRIBUTE);
      if (originalFill != null) el.setAttribute("fill", originalFill);
    }
  });
};

export interface UseRenderParams {
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
  // The markup of the mounted element, or of markup that failed to parse.
  prevSvg: string | undefined;
  // The state whose colors are on the element, so the next state can undo them.
  prevState: schematic.symbol.State | undefined;
}

const createRenderState = (): RenderState => ({
  svgElement: null,
  baseDims: { width: 0, height: 0 },
  prevSvg: undefined,
  prevState: undefined,
});

const selectState = ({
  activeState,
  spec,
  stateOverrides,
}: UseRenderParams): schematic.symbol.State | undefined => {
  const index = activeState === "active" ? 1 : 0;
  return stateOverrides?.[index] ?? spec?.states[index];
};

/**
 * @returns the root element of svg, or null when the markup is empty, malformed, or
 * not rooted in an svg element. Nothing can mount for a null result, so callers show a
 * warning in its place.
 */
export const parseSVG = (svg: string): SVGSVGElement | null => {
  if (svg.length === 0) return null;
  const root = new DOMParser().parseFromString(svg, "image/svg+xml").documentElement;
  if (root.tagName.toLowerCase() !== "svg" || root.querySelector("parsererror") != null)
    return null;
  return root as unknown as SVGSVGElement;
};

/** @returns whether svg is markup the renderer can mount. See {@link parseSVG}. */
export const isValidSVG = (svg: string): boolean => parseSVG(svg) != null;

// buildSVG records the parsed element's base dimensions, ensures its contents are
// wrapped in a single <g>, mounts it into the container, and notifies onMount. The
// element starts from raw markup, so every derived attribute (state colors, dimensions,
// stroke scaling) must be re-applied by the caller afterwards.
const buildSVG = (
  container: HTMLElement,
  state: RenderState,
  svgElement: SVGSVGElement,
  onMount?: (svgElement: SVGSVGElement) => void,
) => {
  if (state.svgElement != null) {
    state.svgElement.remove();
    state.svgElement = null;
  }
  const doc = svgElement.ownerDocument;
  state.svgElement = svgElement;

  const viewBoxAttr = svgElement.getAttribute("viewBox");
  if (viewBoxAttr) {
    const [, , width, height] = viewBoxAttr.split(" ").map(Number);
    state.baseDims = { width, height };
  } else if (svgElement.viewBox?.baseVal)
    state.baseDims = {
      width: svgElement.viewBox.baseVal.width,
      height: svgElement.viewBox.baseVal.height,
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
  onMount?.(svgElement);
};

const applyScale = (
  state: RenderState,
  orientation: location.Outer,
  scale: number,
  externalScale: number,
) => {
  if (state.svgElement == null) return;
  let preScaledDims = state.baseDims;
  if (direction.construct(orientation) === "y")
    preScaledDims = dimensions.swap(preScaledDims);
  const scaledDims = dimensions.scale(
    preScaledDims,
    Primitive.resolveScale(scale) * Primitive.resolveScale(externalScale),
  );
  state.svgElement.setAttribute("width", scaledDims.width.toString());
  state.svgElement.setAttribute("height", scaledDims.height.toString());
  state.svgElement.setAttribute(
    "viewBox",
    `0 0 ${preScaledDims.width} ${preScaledDims.height}`,
  );
};

const applyStrokeScaled = (state: RenderState, strokeScaled: boolean) => {
  if (state.svgElement == null) return;
  const pathElements = state.svgElement.querySelectorAll(
    "path, circle, rect, line, ellipse, polygon, polyline",
  );
  if (!strokeScaled)
    pathElements.forEach((el) =>
      el.setAttribute("vector-effect", "non-scaling-stroke"),
    );
  else pathElements.forEach((el) => el.removeAttribute("vector-effect"));
};

const runRender = (
  container: HTMLElement,
  params: UseRenderParams,
  state: RenderState,
) => {
  const { orientation, externalScale, spec, onMount } = params;
  if (spec == null) return;
  if (state.prevSvg !== spec.svg) {
    state.prevSvg = spec.svg;
    state.prevState = undefined;
    const root = parseSVG(spec.svg);
    if (root == null) {
      state.svgElement?.remove();
      state.svgElement = null;
      return;
    }
    buildSVG(container, state, root, onMount);
  }
  // Markup that failed to parse mounts nothing until it changes.
  if (state.svgElement == null) return;
  const currState = selectState(params);
  if (currState != null) {
    applyState(state.svgElement, currState, state.prevState);
    state.prevState = currState;
  }
  applyScale(state, orientation, spec.scale, externalScale);
  applyStrokeScaled(state, spec.strokeScaled);
};

/// useRender returns a ref callback that drives the SVG mount/state/scale lifecycle for
/// a custom symbol. The returned callback is stable across renders. When the container
/// element attaches, the SVG is built and inserted; when it detaches, the SVG is
/// removed and internal state is cleared so the next attach re-creates the SVG cleanly
/// (including after a Missing→Resolved→Missing→Resolved cycle and under StrictMode's
/// simulated remount). The SVG updates when a param it draws changes identity, so copy
/// the spec on change and never change it in place.
export const useRender = (params: UseRenderParams): RefCallback<HTMLElement> => {
  const { orientation, externalScale, spec } = params;
  const containerRef = useRef<HTMLElement | null>(null);
  const paramsRef = useSyncedRef(params);
  const stateRef = useInitializerRef<RenderState>(createRenderState);
  const currState = selectState(params);

  useLayoutEffect(() => {
    if (containerRef.current != null)
      runRender(containerRef.current, paramsRef.current, stateRef.current);
  }, [
    paramsRef,
    stateRef,
    orientation,
    externalScale,
    spec?.svg,
    spec?.scale,
    spec?.strokeScaled,
    currState,
  ]);

  return (el) => {
    if (el == null) {
      const { svgElement } = stateRef.current;
      if (svgElement != null) svgElement.remove();
      stateRef.current = createRenderState();
      containerRef.current = null;
      return;
    }
    containerRef.current = el;
    runRender(el, paramsRef.current, stateRef.current);
  };
};
