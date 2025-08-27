import { type schematic } from "@synnaxlabs/client";
import { deep, dimensions, direction, type location } from "@synnaxlabs/x";
import { useRef } from "react";

const applyState = (
  svgElement: Element,
  state: schematic.symbol.State,
  prevState?: schematic.symbol.State | null,
) => {
  if (prevState != null)
    prevState.regions.forEach((region) => {
      region.selectors.forEach((selector) => {
        const elements = svgElement.querySelectorAll(selector);
        elements.forEach((el) => {
          const prevStroke = el.getAttribute("data-original-stroke");
          const prevFill = el.getAttribute("data-original-fill");
          if (prevStroke != null) el.setAttribute("stroke", prevStroke);
          if (prevFill != null) el.setAttribute("fill", prevFill);
        });
      });
    });

  state.regions.forEach((region) => {
    region.selectors.forEach((selector) => {
      const elements = svgElement.querySelectorAll(selector);
      elements.forEach((el) => {
        const prevStroke = el.getAttribute("stroke");
        const prevFill = el.getAttribute("fill");
        if (region.strokeColor != null) el.setAttribute("stroke", region.strokeColor);
        if (region.fillColor) el.setAttribute("fill", region.fillColor);
        if (!el.hasAttribute("data-original-stroke") && prevStroke != null)
          el.setAttribute("data-original-stroke", prevStroke);
        if (!el.hasAttribute("data-original-fill") && prevFill != null)
          el.setAttribute("data-original-fill", prevFill);
      });
    });
  });
};

export const useApplyRemote = (
  container: HTMLElement | null,
  orientation: location.Outer,
  activeState: string,
  externalScale: number,
  spec?: schematic.symbol.Spec,
  onMount?: (svgElement: SVGSVGElement) => void,
) => {
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
    baseDimsRef.current = {
      width: svgElementRef.current.viewBox.baseVal.width,
      height: svgElementRef.current.viewBox.baseVal.height,
    };
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

  if (internalScaleDiffers || externalScaleDiffers || orientationDiffers) {
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
      pathElements.forEach((el) => {
        el.setAttribute("vector-effect", "non-scaling-stroke");
      });
    else
      pathElements.forEach((el) => {
        el.removeAttribute("vector-effect");
      });
  }
};
