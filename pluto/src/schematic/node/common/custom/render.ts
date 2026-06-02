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
  prevSvg: string | undefined;
  prevScale: number | undefined;
  prevScaleStroke: boolean | undefined;
  prevState: schematic.symbol.State | undefined;
  prevStateOverrides: schematic.symbol.State[] | undefined;
}

const createRenderState = (): RenderState => ({
  svgElement: null,
  baseDims: { width: 0, height: 0 },
  prevExternalScale: undefined,
  prevOrientation: undefined,
  prevSvg: undefined,
  prevScale: undefined,
  prevScaleStroke: undefined,
  prevState: undefined,
  prevStateOverrides: undefined,
});

// buildSVG parses the spec's markup into a fresh SVG element, records its base
// dimensions, ensures its contents are wrapped in a single <g>, mounts it into the
// container, and notifies onMount. The returned element starts from raw markup, so
// every derived attribute (state colors, dimensions, stroke scaling) must be
// re-applied by the caller afterwards.
const buildSVG = (
  container: HTMLElement,
  state: RenderState,
  spec: schematic.symbol.Spec,
  onMount?: (svgElement: SVGSVGElement) => void,
) => {
  if (state.svgElement != null) {
    state.svgElement.remove();
    state.svgElement = null;
  }
  const parser = new DOMParser();
  const doc = parser.parseFromString(spec.svg, "image/svg+xml");
  const svgElement = doc.documentElement as unknown as SVGSVGElement;
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
  const scaledDims = dimensions.scale(preScaledDims, scale * externalScale);
  state.svgElement.setAttribute("width", scaledDims.width.toString());
  state.svgElement.setAttribute("height", scaledDims.height.toString());
  state.svgElement.setAttribute(
    "viewBox",
    `0 0 ${preScaledDims.width} ${preScaledDims.height}`,
  );
};

const applyScaleStroke = (state: RenderState, scaleStroke: boolean) => {
  if (state.svgElement == null) return;
  const pathElements = state.svgElement.querySelectorAll(
    "path, circle, rect, line, ellipse, polygon, polyline",
  );
  if (!scaleStroke)
    pathElements.forEach((el) =>
      el.setAttribute("vector-effect", "non-scaling-stroke"),
    );
  else pathElements.forEach((el) => el.removeAttribute("vector-effect"));
};

const runRender = (container: HTMLElement, args: UseRenderArgs, state: RenderState) => {
  const { orientation, activeState, externalScale, spec, onMount, stateOverrides } =
    args;
  if (spec == null || spec.svg.length === 0) return;

  // useRender has two callers with opposite mutation models: the schematic node
  // renderers receive a fresh spec reference from the flux cache on every update,
  // while the symbol editor's form mutates a single spec object in place. Diffing by
  // object identity is therefore wrong - it never detects the editor's in-place
  // edits. Compare against value snapshots instead: primitives by value, states by
  // deep equality.
  const externalScaleDiffers = state.prevExternalScale !== externalScale;
  const orientationDiffers = state.prevOrientation !== orientation;
  const svgDiffers = state.prevSvg !== spec.svg;
  const scaleDiffers = state.prevScale !== spec.scale;
  const scaleStrokeDiffers = state.prevScaleStroke !== spec.scaleStroke;

  const stateIndex = activeState === "active" ? 1 : 0;
  const currState = stateOverrides?.[stateIndex] ?? spec.states[stateIndex];

  const stateDiffers = !deep.equal(state.prevState, currState);
  const stateOverridesDiffers = !deep.equal(state.prevStateOverrides, stateOverrides);

  if (
    !externalScaleDiffers &&
    !orientationDiffers &&
    !svgDiffers &&
    !scaleDiffers &&
    !scaleStrokeDiffers &&
    !stateDiffers &&
    !stateOverridesDiffers
  )
    return;

  // A rebuilt SVG starts from raw markup with none of its derived attributes set, so
  // colors, dimensions, and stroke scaling must all be re-applied even when their own
  // inputs are unchanged.
  const rebuilt = state.svgElement == null || svgDiffers;
  if (rebuilt) buildSVG(container, state, spec, onMount);

  if (currState != null && (rebuilt || stateDiffers || stateOverridesDiffers)) {
    applyState(state.svgElement!, currState, rebuilt ? undefined : state.prevState);
    state.prevState = deep.copy(currState);
  }

  if (rebuilt || scaleDiffers || externalScaleDiffers || orientationDiffers)
    applyScale(state, orientation, spec.scale, externalScale);

  if (rebuilt || scaleStrokeDiffers) applyScaleStroke(state, spec.scaleStroke);

  state.prevExternalScale = externalScale;
  state.prevOrientation = orientation;
  state.prevSvg = spec.svg;
  state.prevScale = spec.scale;
  state.prevScaleStroke = spec.scaleStroke;
  state.prevStateOverrides = deep.copy(stateOverrides);
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
