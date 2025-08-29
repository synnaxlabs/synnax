import { type schematic } from "@synnaxlabs/client";
import { deep, dimensions, direction, type location } from "@synnaxlabs/x";
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
  iterElements(state, svgElement, (el, { strokeColor, fillColor }) => {
    storeOriginalAttributes(el);
    if (strokeColor != null) el.setAttribute("stroke", strokeColor);
    else {
      // Restore original stroke if no strokeColor specified
      const originalStroke = el.getAttribute(ORIGINAL_STROKE_ATTRIBUTE);
      if (originalStroke != null) el.setAttribute("stroke", originalStroke);
    }
    
    if (fillColor != null) el.setAttribute("fill", fillColor);
    else {
      // Restore original fill if no fillColor specified
      const originalFill = el.getAttribute(ORIGINAL_FILL_ATTRIBUTE);
      if (originalFill != null) el.setAttribute("fill", originalFill);
    }
  });
};

export interface UseCustomArgs {
  container: HTMLElement | null;
  orientation: location.Outer;
  activeState: string;
  externalScale: number;
  spec?: schematic.symbol.Spec;
  onMount?: (svgElement: SVGSVGElement) => void;
}

export const useCustom = ({
  container,
  orientation,
  activeState,
  externalScale,
  spec,
  onMount,
}: UseCustomArgs) => {
  const svgElementRef = useRef<SVGSVGElement>(null);
  const baseDimsRef = useRef<dimensions.Dimensions>({ width: 0, height: 0 });

  const prevExternalScaleRef = useRef<number | undefined>(undefined);
  const prevOrientationRef = useRef<location.Outer | undefined>(undefined);
  const prevSpecDataRef = useRef<schematic.symbol.Spec | undefined>(undefined);
  const prevStateRef = useRef<schematic.symbol.State>(undefined);

  if (spec == null || spec.svg.length === 0 || container == null) return;

  const externalScaleDiffers = prevExternalScaleRef.current !== externalScale;
  const svgDiffers = prevSpecDataRef.current?.svg !== spec?.svg;
  const orientationDiffers = prevOrientationRef.current !== orientation;
  const internalScaleDiffers = prevSpecDataRef.current?.scale !== spec?.scale;
  const scaleStrokeDiffers = prevSpecDataRef.current?.scaleStroke !== spec?.scaleStroke;
  const specDiffers = prevSpecDataRef.current !== spec;
  const currState = activeState === "active" ? spec.states[1] : spec.states[0];
  const stateDiffers = prevStateRef.current !== currState;
  const different =
    externalScaleDiffers || svgDiffers || scaleStrokeDiffers || stateDiffers;
  if (!different) return;
  if (externalScaleDiffers) prevExternalScaleRef.current = externalScale;
  if (orientationDiffers) prevOrientationRef.current = orientation;
  if (specDiffers) prevSpecDataRef.current = deep.copy(spec);
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

    // Extract dimensions from viewBox attribute for better test compatibility
    const viewBoxAttr = svgElementRef.current.getAttribute("viewBox");
    if (viewBoxAttr) {
      const [, , width, height] = viewBoxAttr.split(" ").map(Number);
      baseDimsRef.current = { width, height };
    } else if (svgElementRef.current.viewBox?.baseVal)
      baseDimsRef.current = {
        width: svgElementRef.current.viewBox.baseVal.width,
        height: svgElementRef.current.viewBox.baseVal.height,
      };
    else
      // Fallback to default dimensions if viewBox is not available
      baseDimsRef.current = { width: 100, height: 100 };

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

  if (stateDiffers) {
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
    // Use direction.construct to properly determine if we need to swap
    // This handles the rotation logic correctly
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
