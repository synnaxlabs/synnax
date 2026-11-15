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
import { color, dimensions, direction, id, location } from "@synnaxlabs/x";
import { type RefCallback, useLayoutEffect, useRef } from "react";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";
import { Region } from "@/schematic/node/common/region";

const ORIGINAL_FILL_ATTRIBUTE = "data-original-fill";
const SVG_NAMESPACE = "http://www.w3.org/2000/svg";

const createSVGElement = <K extends keyof SVGElementTagNameMap>(
  doc: Document,
  tag: K,
  attributes: Record<string, string | number>,
): SVGElementTagNameMap[K] => {
  const el = doc.createElementNS(SVG_NAMESPACE, tag);
  Object.entries(attributes).forEach(([k, v]) => el.setAttribute(k, v.toString()));
  return el;
};

const iterElements = (
  state: schematic.symbol.State,
  drawing: Element,
  fn: (el: SVGElement, region: schematic.symbol.Region) => void,
) => {
  state.regions.forEach((region) => {
    region.selectors.forEach((selector) => {
      const elements = drawing.querySelectorAll<SVGElement>(selector);
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

const applyState = (
  drawing: Element,
  state: schematic.symbol.State,
  prevState?: schematic.symbol.State | null,
) => {
  if (prevState != null) iterElements(prevState, drawing, clearState);
  iterElements(state, drawing, (el, { strokeColor, fillColor }) => {
    setStroke(el, strokeColor);
    setFill(el, fillColor);
  });
};

// The root carries the band class as on built-in symbols, so rules that read the
// display color, such as the toggle states, apply to the drawing.
const applySymbolColor = (svgElement: SVGSVGElement, symbolColor?: color.Crude) => {
  svgElement.classList.add(CSS.symbolColored, CSS.B("custom-drawing"));
  const rgba = color.rgbaString(symbolColor);
  if (rgba == null) svgElement.style.removeProperty(Primitive.SYMBOL_COLOR);
  else svgElement.style.setProperty(Primitive.SYMBOL_COLOR, rgba);
};

const isStoredStrokeShown = (el: Element): boolean => {
  const stroke = el.getAttribute("stroke");
  if (stroke == null || stroke === "none") return false;
  const parsed = color.colorZ.safeParse(stroke);
  return !parsed.success || color.opacity(parsed.data) > 0;
};

// A symbol color, such as the stale color, replaces each stroke the drawing shows: the
// stroke of the shape's region, or its stored stroke when no region sets one. It runs
// after applyState and overrides it.
const applySymbolStrokes = (
  drawing: Element,
  state: schematic.symbol.State | undefined,
  symbolColor: color.Crude | undefined,
) => {
  const regionStrokes = new Map<Element, color.Color>();
  if (state != null)
    iterElements(state, drawing, (el, { strokeColor }) => {
      if (strokeColor != null) regionStrokes.set(el, strokeColor);
    });
  Region.collectVisualElements(drawing).forEach((el) => {
    const regionStroke = regionStrokes.get(el);
    const shown =
      regionStroke != null ? color.opacity(regionStroke) > 0 : isStoredStrokeShown(el);
    if (symbolColor != null && shown) setStroke(el, symbolColor);
    else if (regionStroke == null) setStroke(el, undefined);
  });
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
  /** True in a delayed toggle, so the drawing fills inside its shapes while held. */
  delayed?: boolean;
}

interface RenderState {
  svgElement: SVGSVGElement | null;
  // The <g> that holds the drawing's own shapes, so region selectors cannot reach the
  // hold fill.
  drawing: SVGGElement | null;
  baseDims: dimensions.Dimensions;
  // The markup of the mounted element, or of markup that failed to parse.
  prevSvg: string | undefined;
  prevDelayed: boolean;
  // The state whose colors are on the element, so the next state can undo them.
  prevState: schematic.symbol.State | undefined;
}

const createRenderState = (): RenderState => ({
  svgElement: null,
  drawing: null,
  baseDims: { width: 0, height: 0 },
  prevSvg: undefined,
  prevDelayed: false,
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

// The hold fill paints inside the drawing's shapes through a mask, as on built-in
// symbols. Returns the new <g> that holds the drawing.
const appendHoldFill = (g: SVGGElement, dims: dimensions.Dimensions): SVGGElement => {
  const doc = g.ownerDocument;
  const key = id.create();
  const drawing = createSVGElement(doc, "g", { id: key });
  while (g.firstChild != null) drawing.appendChild(g.firstChild);
  const box = Primitive.holdBox(dims);
  const mask = createSVGElement(doc, "mask", {
    id: `${key}-mask`,
    class: CSS.BE("symbol-hold", "mask"),
    maskUnits: "userSpaceOnUse",
    ...box,
  });
  mask.append(createSVGElement(doc, "use", { href: `#${key}` }));
  const fill = createSVGElement(doc, "rect", {
    class: CSS.BE("symbol-hold", "fill"),
    mask: `url(#${key}-mask)`,
    ...box,
  });
  g.append(drawing, mask, fill);
  return drawing;
};

// buildSVG records the parsed element's base dimensions, wraps its contents in one new
// <g> for the stylesheet to turn, adds the hold fill when delayed, mounts it into the
// container, and notifies onMount. The element starts from raw markup, so every derived
// attribute (state colors, dimensions, orientation, stroke scaling) must be re-applied
// by the caller afterwards.
const buildSVG = (
  container: HTMLElement,
  state: RenderState,
  svgElement: SVGSVGElement,
  delayed: boolean,
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

  const g = createSVGElement(doc, "g", {});
  while (svgElement.firstChild != null) g.appendChild(svgElement.firstChild);
  state.drawing = delayed ? appendHoldFill(g, state.baseDims) : g;
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

const applyStrokeScaled = (drawing: Element, strokeScaled: boolean) => {
  const pathElements = drawing.querySelectorAll(
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
  const {
    orientation,
    externalScale,
    spec,
    onMount,
    strokeColor,
    delayed = false,
  } = params;
  if (spec == null) return;
  if (state.prevSvg !== spec.svg || state.prevDelayed !== delayed) {
    state.prevSvg = spec.svg;
    state.prevDelayed = delayed;
    state.prevState = undefined;
    const root = Region.parseSVG(spec.svg);
    if (root == null) {
      state.svgElement?.remove();
      state.svgElement = null;
      state.drawing = null;
      return;
    }
    buildSVG(container, state, root, delayed, onMount);
  }
  const { svgElement, drawing } = state;
  // Markup that failed to parse mounts nothing until it changes.
  if (svgElement == null || drawing == null) return;
  applySymbolColor(svgElement, strokeColor);
  const currState = selectState(params);
  if (currState != null) {
    applyState(drawing, currState, state.prevState);
    state.prevState = currState;
  }
  applySymbolStrokes(drawing, currState, strokeColor);
  applyScale(state, orientation, spec.scale, externalScale);
  applyOrientation(svgElement, orientation);
  applyStrokeScaled(drawing, spec.strokeScaled);
};

/// useRender returns a ref callback that drives the SVG mount/state/scale lifecycle for
/// a custom symbol. The returned callback is stable across renders. When the container
/// element attaches, the SVG is built and inserted; when it detaches, the SVG is
/// removed and internal state is cleared so the next attach re-creates the SVG cleanly
/// (including after a Missing→Resolved→Missing→Resolved cycle and under StrictMode's
/// simulated remount). The SVG updates when a param it draws changes identity, so copy
/// the spec on change and never change it in place.
export const useRender = (params: UseRenderParams): RefCallback<HTMLElement> => {
  const { orientation, externalScale, spec, strokeColor, delayed } = params;
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
    delayed,
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
