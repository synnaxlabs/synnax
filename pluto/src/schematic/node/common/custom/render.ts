// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { useInitializerRef, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { color } from "@synnaxlabs/x";
import { type RefCallback, useLayoutEffect, useRef } from "react";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";
import { Region } from "@/schematic/node/common/region";

const ORIGINAL_FILL_ATTRIBUTE = "data-original-fill";
const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
const XMLNS_NAMESPACE = "http://www.w3.org/2000/xmlns/";
// Primitive.SVG sizes the drawing, and the symbol CSS overrides the fill and stroke of
// the root, so these root attributes stay behind.
const ROOT_ONLY_ATTRIBUTES = new Set([
  "width",
  "height",
  "viewBox",
  "x",
  "y",
  "preserveAspectRatio",
  "fill",
  "stroke",
]);

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
  activeState: string;
  spec?: schematic.symbol.Spec;
  /** Called with the <g> that holds the drawing each time the drawing mounts. */
  onMount?: (drawing: SVGGElement) => void;
  stateOverrides?: schematic.symbol.State[];
  /** The color of the whole symbol, such as the stale color. */
  strokeColor?: color.Crude;
  /** Marks markup that ships with Pluto, which mounts without the sanitizer. */
  builtIn?: boolean;
}

interface RenderState {
  drawing: SVGGElement | null;
  // The markup of the mounted drawing, or of markup that failed to parse.
  prevSvg: string | undefined;
  // The state whose colors are on the drawing, so the next state can undo them.
  prevState: schematic.symbol.State | undefined;
}

const createRenderState = (): RenderState => ({
  drawing: null,
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
 * @returns whether svg is markup the renderer can mount. See {@link Region.measure}.
 */
export const isValidSVG = (svg: string): boolean => Region.measure(svg) != null;

// buildDrawing moves the contents of root into one new <g> in the host. The <g> takes
// the other root attributes, such as stroke-width. The caller re-applies state colors
// and stroke scaling afterwards.
const buildDrawing = (
  host: Element,
  state: RenderState,
  root: SVGSVGElement,
  onMount?: (drawing: SVGGElement) => void,
) => {
  state.drawing?.remove();
  const drawing = root.ownerDocument.createElementNS(SVG_NAMESPACE, "g");
  for (const { namespaceURI, name, value } of Array.from(root.attributes))
    if (namespaceURI !== XMLNS_NAMESPACE && !ROOT_ONLY_ATTRIBUTES.has(name))
      drawing.setAttributeNS(namespaceURI, name, value);
  while (root.firstChild != null) drawing.appendChild(root.firstChild);
  host.appendChild(drawing);
  state.drawing = drawing;
  onMount?.(drawing);
};

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

const runRender = (host: Element, params: UseRenderParams, state: RenderState) => {
  const { spec, onMount, strokeColor, builtIn = false } = params;
  if (spec == null) return;
  if (state.prevSvg !== spec.svg) {
    state.prevSvg = spec.svg;
    state.prevState = undefined;
    const root = builtIn ? Region.parseTrusted(spec.svg) : Region.parseSVG(spec.svg);
    if (root == null) {
      state.drawing?.remove();
      state.drawing = null;
      return;
    }
    buildDrawing(host, state, root, onMount);
  }
  const { drawing } = state;
  // Markup that failed to parse mounts nothing until it changes.
  if (drawing == null) return;
  const currState = selectState(params);
  if (currState != null) {
    applyState(drawing, currState, state.prevState);
    state.prevState = currState;
  }
  applySymbolStrokes(drawing, currState, strokeColor);
  applyStrokeScaled(drawing, spec.strokeScaled);
};

/// useRender returns a ref callback that mounts the drawing of a custom symbol into the
/// element it attaches to, as one <g>, and keeps its state colors and strokes applied.
/// Attach it inside an <svg> that sizes and turns the drawing, such as Primitive.SVG.
/// The returned callback is stable across renders. When the element detaches, the
/// drawing is removed and internal state is cleared so the next attach re-creates it
/// cleanly (including after a Missing→Resolved→Missing→Resolved cycle and under
/// StrictMode's simulated remount). The drawing updates when a param it draws changes
/// identity, so copy the spec on change and never change it in place.
export const useRender = (params: UseRenderParams): RefCallback<Element> => {
  const { spec, strokeColor } = params;
  const hostRef = useRef<Element | null>(null);
  const paramsRef = useSyncedRef(params);
  const stateRef = useInitializerRef<RenderState>(createRenderState);
  const currState = selectState(params);
  const strokeRGBA = color.rgbaString(strokeColor);

  useLayoutEffect(() => {
    if (hostRef.current != null)
      runRender(hostRef.current, paramsRef.current, stateRef.current);
  }, [paramsRef, stateRef, spec?.svg, spec?.strokeScaled, currState, strokeRGBA]);

  return (el) => {
    if (el == null) {
      stateRef.current.drawing?.remove();
      stateRef.current = createRenderState();
      hostRef.current = null;
      return;
    }
    hostRef.current = el;
    runRender(el, paramsRef.current, stateRef.current);
  };
};
