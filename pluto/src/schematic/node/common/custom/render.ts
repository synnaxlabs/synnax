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
import { color, dimensions, direction, location } from "@synnaxlabs/x";
import { type RefCallback, useLayoutEffect, useRef } from "react";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";
import { Region } from "@/schematic/node/common/region";

const ORIGINAL_FILL_ATTRIBUTE = "data-original-fill";

const iterElements = (
  state: schematic.symbol.State,
  svgElement: Element,
  fn: (el: SVGElement, region: schematic.symbol.Region) => void,
) => {
  state.regions.forEach((region) => {
    region.selectors.forEach((selector) => {
      const elements = svgElement.querySelectorAll<SVGElement>(selector);
      elements.forEach((el) => fn(el, region));
    });
  });
};

// The theme band paints the stroke, as on built-in symbols. The fill stays as stored,
// as on the tank and the cylinder.
const setStroke = (el: SVGElement, strokeColor: color.Crude | undefined) => {
  const rgba = color.rgbaString(strokeColor);
  if (rgba == null) {
    el.classList.remove(CSS.symbolColored);
    el.style.removeProperty(Primitive.SYMBOL_COLOR);
    return;
  }
  el.classList.add(CSS.symbolColored);
  el.style.setProperty(Primitive.SYMBOL_COLOR, rgba);
};

const setFill = (el: SVGElement, fillColor: color.Color | undefined) => {
  if (!el.hasAttribute(ORIGINAL_FILL_ATTRIBUTE)) {
    const originalFill = el.getAttribute("fill");
    if (originalFill != null) el.setAttribute(ORIGINAL_FILL_ATTRIBUTE, originalFill);
  }
  const fill =
    fillColor != null ? color.hex(fillColor) : el.getAttribute(ORIGINAL_FILL_ATTRIBUTE);
  if (fill != null) el.setAttribute("fill", fill);
};

const clearState = (el: SVGElement) => {
  setStroke(el, undefined);
  setFill(el, undefined);
};

// A symbol color, such as the stale color, replaces each region stroke the drawing
// shows. A region with no stroke stays without one.
const applyState = (
  svgElement: Element,
  state: schematic.symbol.State,
  symbolColor: color.Crude | undefined,
  prevState?: schematic.symbol.State | null,
) => {
  if (prevState != null) iterElements(prevState, svgElement, clearState);
  iterElements(state, svgElement, (el, { strokeColor, fillColor }) => {
    const shown = strokeColor != null && color.opacity(strokeColor) > 0;
    setStroke(el, symbolColor != null && shown ? symbolColor : strokeColor);
    setFill(el, fillColor);
  });
};

// The root carries the band class as on built-in symbols, so rules that read the
// display color, such as the toggle states, apply to the drawing.
const applySymbolColor = (svgElement: SVGSVGElement, symbolColor?: color.Crude) => {
  svgElement.classList.add(CSS.symbolColored);
  const rgba = color.rgbaString(symbolColor);
  if (rgba == null) svgElement.style.removeProperty(Primitive.SYMBOL_COLOR);
  else svgElement.style.setProperty(Primitive.SYMBOL_COLOR, rgba);
};

export interface UseRenderParams {
  orientation: location.Outer;
  activeState: string;
  externalScale: number;
  spec?: schematic.symbol.Spec;
  onMount?: (svgElement: SVGSVGElement) => void;
  stateOverrides?: schematic.symbol.State[];
  /** The color of the whole symbol, such as the stale color. */
  strokeColor?: color.Crude;
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
 * @returns whether svg is markup the renderer can mount. See {@link Region.parseSVG}.
 */
export const isValidSVG = (svg: string): boolean => Region.parseSVG(svg) != null;

// buildSVG records the parsed element's base dimensions, wraps its contents in one new
// <g> for the stylesheet to turn, mounts it into the container, and notifies onMount.
// The element starts from raw markup, so every derived attribute (state colors,
// dimensions, orientation, stroke scaling) must be re-applied by the caller afterwards.
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

  const g = doc.createElementNS("http://www.w3.org/2000/svg", "g");
  while (svgElement.firstChild != null) g.appendChild(svgElement.firstChild);
  svgElement.append(g);
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

// The location class turns the drawing, as on built-in symbols.
const applyOrientation = (svgElement: SVGSVGElement, orientation: location.Outer) =>
  location.OUTER_LOCATIONS.forEach((loc) =>
    svgElement.classList.toggle(CSS.loc(loc), loc === orientation),
  );

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
  const { orientation, externalScale, spec, onMount, strokeColor } = params;
  if (spec == null) return;
  if (state.prevSvg !== spec.svg) {
    state.prevSvg = spec.svg;
    state.prevState = undefined;
    const root = Region.parseSVG(spec.svg);
    if (root == null) {
      state.svgElement?.remove();
      state.svgElement = null;
      return;
    }
    buildSVG(container, state, root, onMount);
  }
  // Markup that failed to parse mounts nothing until it changes.
  if (state.svgElement == null) return;
  applySymbolColor(state.svgElement, strokeColor);
  const currState = selectState(params);
  if (currState != null) {
    applyState(state.svgElement, currState, strokeColor, state.prevState);
    state.prevState = currState;
  }
  applyScale(state, orientation, spec.scale, externalScale);
  applyOrientation(state.svgElement, orientation);
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
  const { orientation, externalScale, spec, strokeColor } = params;
  const containerRef = useRef<HTMLElement | null>(null);
  const paramsRef = useSyncedRef(params);
  const stateRef = useInitializerRef<RenderState>(createRenderState);
  const currState = selectState(params);
  const strokeRGBA = color.rgbaString(strokeColor);

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
    strokeRGBA,
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
