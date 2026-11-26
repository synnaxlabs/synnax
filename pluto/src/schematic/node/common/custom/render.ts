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
import {
  Motion,
  type MotionParams,
  moves,
} from "@/schematic/node/common/custom/motion";
import type { Paint } from "@/schematic/node/common/custom/paint";
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
  paints: Paint[],
  drawing: Element,
  fn: (el: SVGElement, paint: Paint) => void,
) =>
  paints.forEach((paint) => {
    const elements =
      paint.selectors == null
        ? Region.allGraphics(drawing)
        : Region.graphics(drawing, paint.selectors);
    elements.forEach((el) => fn(el, paint));
  });

// Paint colors arrive in the theme band, so the stroke takes the color as it is.
const setStroke = (el: SVGElement, strokeColor: color.Color | undefined) => {
  if (strokeColor == null) el.style.removeProperty("stroke");
  else el.style.setProperty("stroke", color.hex(strokeColor));
};

// The CSS theme band paints a symbol color over the stroke of the paint.
const setSymbolStroke = (el: SVGElement, symbolColor: color.Crude | undefined) => {
  const rgba = color.rgbaString(symbolColor);
  if (rgba == null) {
    el.classList.remove(CSS.symbolColored);
    el.style.removeProperty(Primitive.SYMBOL_COLOR);
    return;
  }
  el.style.removeProperty("stroke");
  el.classList.add(CSS.symbolColored);
  el.style.setProperty(Primitive.SYMBOL_COLOR, rgba);
};

const setFill = (el: SVGElement, fillColor: color.Color | undefined) => {
  // An empty original marks an element with no fill of its own, such as a <use>.
  const original =
    el.getAttribute(ORIGINAL_FILL_ATTRIBUTE) ?? el.getAttribute("fill") ?? "";
  el.setAttribute(ORIGINAL_FILL_ATTRIBUTE, original);
  if (fillColor != null) el.setAttribute("fill", color.hex(fillColor));
  else if (original === "") el.removeAttribute("fill");
  else el.setAttribute("fill", original);
};

const setHidden = (el: SVGElement, hidden: boolean) => {
  if (hidden) el.style.setProperty("display", "none");
  else el.style.removeProperty("display");
};

const clearPaint = (el: SVGElement) => {
  setStroke(el, undefined);
  setFill(el, undefined);
  setHidden(el, false);
};

const isStoredStrokeHidden = (el: Element): boolean => {
  const stroke = el.getAttribute("stroke");
  if (stroke == null) return false;
  if (stroke === "none") return true;
  const parsed = color.colorZ.safeParse(stroke);
  return parsed.success && color.opacity(parsed.data) === 0;
};

const isStoredStrokeShown = (el: Element): boolean =>
  el.hasAttribute("stroke") && !isStoredStrokeHidden(el);

// A whole-drawing stroke skips a shape whose markup hides its stroke. A part stroke
// does not.
const strokeOf = (
  el: Element,
  { stroke, selectors }: Paint,
): color.Color | undefined =>
  stroke == null || (selectors == null && isStoredStrokeHidden(el))
    ? undefined
    : stroke;

// A later paint wins on each field that it sets.
const applyPaints = (drawing: Element, paints: Paint[], prevPaints: Paint[]) => {
  iterElements(prevPaints, drawing, clearPaint);
  iterElements(paints, drawing, (el, paint) => {
    const stroke = strokeOf(el, paint);
    if (stroke != null) setStroke(el, stroke);
    if (paint.fill != null) setFill(el, paint.fill);
    if (paint.hidden != null) setHidden(el, paint.hidden);
  });
};

// A symbol color replaces each stroke the drawing shows: the stroke of the shape's
// paint, or its stored stroke when no paint sets one. It runs after applyPaints and
// overrides it.
const applySymbolStrokes = (
  drawing: Element,
  paints: Paint[],
  symbolColor: color.Crude | undefined,
) => {
  const paintStrokes = new Map<SVGElement, color.Color>();
  iterElements(paints, drawing, (el, paint) => {
    const stroke = strokeOf(el, paint);
    if (stroke != null) paintStrokes.set(el, stroke);
  });
  const elements = new Set([...Region.allGraphics(drawing), ...paintStrokes.keys()]);
  elements.forEach((el) => {
    const paintStroke = paintStrokes.get(el);
    const shown =
      paintStroke != null ? color.opacity(paintStroke) > 0 : isStoredStrokeShown(el);
    setSymbolStroke(el, symbolColor != null && shown ? symbolColor : undefined);
  });
};

export interface UseRenderParams {
  spec?: schematic.symbol.Spec;
  /** paints are the paints that the drawing shows, from usePaints. */
  paints?: Paint[];
  /** Called with the <g> that holds the drawing each time the drawing mounts. */
  onMount?: (drawing: SVGGElement) => void;
  /** The color of the whole symbol. */
  strokeColor?: color.Crude;
  /** Marks markup that ships with Pluto, which mounts without the sanitizer. */
  builtIn?: boolean;
  /** motion moves the parts of the drawing by its looks. With none, nothing moves. */
  motion?: MotionParams;
}

interface RenderState {
  drawing: SVGGElement | null;
  // The markup of the mounted drawing, or of markup that failed to parse.
  prevSvg: string | undefined;
  // The paints on the drawing, so the next paints can undo them.
  prevPaints: Paint[];
  motion: Motion | null;
}

const NO_PAINTS: Paint[] = [];

const createRenderState = (): RenderState => ({
  drawing: null,
  prevSvg: undefined,
  prevPaints: NO_PAINTS,
  motion: null,
});

const stopMotion = (state: RenderState) => {
  state.motion?.stop();
  state.motion = null;
};

const applyMotion = (
  spec: schematic.symbol.Spec,
  params: MotionParams | undefined,
  state: RenderState,
) => {
  const { drawing } = state;
  if (drawing == null) return;
  if (params == null || state.motion?.loop !== params.loop) stopMotion(state);
  if (params == null) return;
  if (state.motion == null) {
    if (!spec.looks.some(moves)) return;
    const box = Region.measure(spec.drawing.markup);
    if (box == null) return;
    state.motion = new Motion(params.loop, drawing, box);
  }
  state.motion.update(spec, params);
};

/**
 * @returns whether svg is markup the renderer can mount. See {@link Region.measure}.
 */
export const isValidSVG = (svg: string): boolean => Region.measure(svg) != null;

// Motion writes these as presentation attributes, which inline declarations outrank.
const MOTION_PROPERTIES = ["opacity", "stroke-dashoffset"];

const moveMotionStyles = (drawing: SVGGElement) => {
  for (const el of [drawing, ...drawing.querySelectorAll<SVGElement>("[style]")])
    for (const property of MOTION_PROPERTIES) {
      const value = el.style.getPropertyValue(property);
      if (value === "") continue;
      el.setAttribute(property, value);
      el.style.removeProperty(property);
    }
};

// buildDrawing moves the contents of root into one new <g> in the host. The <g> takes
// the other root attributes, such as stroke-width. The caller re-applies state colors
// and stroke scaling afterwards.
const buildDrawing = (
  host: Element,
  state: RenderState,
  root: SVGSVGElement,
  onMount?: (drawing: SVGGElement) => void,
) => {
  stopMotion(state);
  state.drawing?.remove();
  const drawing = root.ownerDocument.createElementNS(SVG_NAMESPACE, "g");
  for (const { namespaceURI, name, value } of Array.from(root.attributes))
    if (namespaceURI !== XMLNS_NAMESPACE && !ROOT_ONLY_ATTRIBUTES.has(name))
      drawing.setAttributeNS(namespaceURI, name, value);
  while (root.firstChild != null) drawing.appendChild(root.firstChild);
  moveMotionStyles(drawing);
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
  const { spec, paints = NO_PAINTS, onMount, strokeColor, builtIn = false } = params;
  if (spec == null) return;
  const { markup } = spec.drawing;
  if (state.prevSvg !== markup) {
    state.prevSvg = markup;
    state.prevPaints = NO_PAINTS;
    const root = builtIn ? Region.parseTrusted(markup) : Region.parseSVG(markup);
    if (root == null) {
      stopMotion(state);
      state.drawing?.remove();
      state.drawing = null;
      return;
    }
    buildDrawing(host, state, root, onMount);
  }
  const { drawing } = state;
  // Markup that failed to parse mounts nothing until it changes.
  if (drawing == null) return;
  applyPaints(drawing, paints, state.prevPaints);
  state.prevPaints = paints;
  applySymbolStrokes(drawing, paints, strokeColor);
  applyStrokeScaled(drawing, spec.strokeScaled);
  applyMotion(spec, params.motion, state);
};

/// useRender returns a ref callback that mounts the drawing of a custom symbol into the
/// element it attaches to, as one <g>, and keeps its paints and strokes applied.
/// Attach it inside an <svg> that sizes and turns the drawing, such as Primitive.SVG.
/// The returned callback is stable across renders. When the element detaches, the
/// drawing is removed and internal state is cleared so the next attach re-creates it
/// cleanly (including after a Missing→Resolved→Missing→Resolved cycle and under
/// StrictMode's simulated remount). The drawing updates when a param it draws changes
/// identity, so copy the spec on change and never change it in place.
export const useRender = (params: UseRenderParams): RefCallback<Element> => {
  const { spec, paints, strokeColor, motion } = params;
  const hostRef = useRef<Element | null>(null);
  const paramsRef = useSyncedRef(params);
  const stateRef = useInitializerRef<RenderState>(createRenderState);
  const strokeRGBA = color.rgbaString(strokeColor);

  useLayoutEffect(() => {
    if (hostRef.current != null)
      runRender(hostRef.current, paramsRef.current, stateRef.current);
  }, [
    paramsRef,
    stateRef,
    spec?.drawing.markup,
    spec?.strokeScaled,
    paints,
    strokeRGBA,
  ]);

  useLayoutEffect(() => {
    const { spec, motion } = paramsRef.current;
    if (spec != null) applyMotion(spec, motion, stateRef.current);
  }, [paramsRef, stateRef, spec?.looks, spec?.parts, spec?.states, motion]);

  return (el) => {
    if (el == null) {
      stopMotion(stateRef.current);
      stateRef.current.drawing?.remove();
      stateRef.current = createRenderState();
      hostRef.current = null;
      return;
    }
    hostRef.current = el;
    runRender(el, paramsRef.current, stateRef.current);
  };
};
