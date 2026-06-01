// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic, ValidationError } from "@synnaxlabs/client";
import { color, id } from "@synnaxlabs/x";

/// VISUAL_ELEMENTS are the SVG tags that carry paint (fill/stroke) and therefore
/// participate in region extraction and color normalization.
export const VISUAL_ELEMENTS = [
  "path",
  "rect",
  "circle",
  "ellipse",
  "polygon",
  "polyline",
  "line",
  "text",
];

// Tags whose descendants define geometry/paint for reuse but are never themselves
// painted. Shapes inside them must be ignored, otherwise clip paths, gradient stops,
// and symbol templates surface as phantom regions. Compared by exact case because SVG
// element names are case-sensitive.
const NON_RENDERED_CONTAINERS = new Set([
  "defs",
  "clipPath",
  "mask",
  "symbol",
  "marker",
  "pattern",
]);

const isRendered = (el: Element): boolean => {
  for (let p = el.parentElement; p != null; p = p.parentElement)
    if (NON_RENDERED_CONTAINERS.has(p.tagName)) return false;
  return true;
};

/// collectVisualElements returns every painted shape in the tree, skipping shapes that
/// live inside non-rendering containers (defs, clipPath, mask, ...).
const collectVisualElements = (svg: Element): SVGElement[] =>
  VISUAL_ELEMENTS.flatMap((tag) =>
    Array.from(svg.querySelectorAll<SVGElement>(tag)),
  ).filter(isRendered);

interface ColorPair {
  stroke: string;
  fill: string;
}

interface ElementGroup {
  elements: SVGElement[];
  colorPair: ColorPair;
}

const TRANSPARENT = color.hex(color.ZERO);

const normalizeColor = (colorStr: string | null | undefined): string => {
  if (colorStr == null || colorStr === "") return TRANSPARENT;
  const parsed = color.fromCSS(colorStr);
  if (parsed == null) return TRANSPARENT;
  return color.hex(parsed);
};

const getElementColors = (element: SVGElement): ColorPair => {
  const stroke = element.style.stroke || element.getAttribute("stroke");
  const fill = element.style.fill || element.getAttribute("fill");
  return { stroke: normalizeColor(stroke), fill: normalizeColor(fill) };
};

const getColorPairKey = (colorPair: ColorPair): string =>
  `${colorPair.stroke}|${colorPair.fill}`;

const generateSelector = (element: SVGElement): string => {
  if (element.id) return `#${element.id}`;
  const dataRegionId = element.getAttribute("data-region-id");
  if (dataRegionId) return `[data-region-id="${dataRegionId}"]`;
  if (element.classList.length > 0)
    return `.${Array.from(element.classList).join(".")}`;
  const tagName = element.tagName.toLowerCase();
  const parent = element.parentElement;
  if (!parent) return tagName;
  let index = 1;
  let sibling = element.previousElementSibling;
  while (sibling) {
    if (sibling.tagName.toLowerCase() === tagName) index++;
    sibling = sibling.previousElementSibling;
  }
  const parentSelector =
    parent.tagName.toLowerCase() === "svg"
      ? ""
      : parent instanceof SVGElement
        ? `${generateSelector(parent)} > `
        : "";
  return `${parentSelector}${tagName}:nth-of-type(${index})`;
};

export const extract = (svgElement: SVGElement): schematic.symbol.Region[] => {
  const visualElements = collectVisualElements(svgElement);
  const groups = new Map<string, ElementGroup>();
  visualElements.forEach((element) => {
    const colorPair = getElementColors(element);
    const key = getColorPairKey(colorPair);
    if (!groups.has(key))
      groups.set(key, {
        elements: [],
        colorPair,
      });
    groups.get(key)!.elements.push(element);
  });
  return Array.from(groups.values()).map((group, index) => {
    const selectors = group.elements.map(generateSelector);
    const uniqueSelectors = Array.from(new Set(selectors));
    const regionIndex = index + 1;
    const region: schematic.symbol.Region = {
      key: `region-${regionIndex}`,
      name: `Region ${regionIndex}`,
      selectors: uniqueSelectors,
    };
    region.strokeColor = group.colorPair.stroke;
    region.fillColor = group.colorPair.fill;
    return region;
  });
};

const REGION_ID_ATTRIBUTE = "data-region-id";

/// SymbolColors holds an element's resolved fill and stroke, each expressed as the
/// value to write to the corresponding presentation attribute. A null value leaves
/// that attribute untouched.
export interface SymbolColors {
  fill: string | null;
  stroke: string | null;
}

/// ColorResolver computes the effective fill and stroke of an element. The default
/// resolver used by normalizeSVG reads the browser's cascade via getComputedStyle;
/// tests inject a deterministic resolver to exercise the rewrite without a browser.
export type ColorResolver = (el: SVGElement) => SymbolColors;

const setPaint = (el: SVGElement, attr: "fill" | "stroke", value: string | null) => {
  if (value == null) return;
  el.setAttribute(attr, value);
  // Drop the now-redundant inline declaration so the presentation attribute is the
  // single source of truth and nothing outranks a later attribute override.
  el.style.removeProperty(attr);
  if (el.getAttribute("style") === "") el.removeAttribute("style");
};

const addRegionIDs = (el: Element) => {
  if (!(el instanceof SVGElement) || el.tagName === "svg") return;
  if (el.id.length === 0 && el.getAttribute(REGION_ID_ATTRIBUTE) == null)
    el.setAttribute(REGION_ID_ATTRIBUTE, `region-${id.create()}`);
  Array.from(el.children).forEach(addRegionIDs);
};

/// normalizeElement rewrites an SVG in place into the canonical symbol form: every
/// painted element's effective fill and stroke (resolved via resolveColors) are
/// flattened onto presentation attributes, <style> blocks are removed, and each shape
/// without an id is tagged with a stable data-region-id. The result is
/// self-contained: its colors no longer depend on inline styles, <style> rules, or
/// ancestor inheritance, so extract and the renderer's color override operate on a
/// single predictable source. The transform is idempotent.
export const normalizeElement = (
  svg: SVGSVGElement,
  resolveColors: ColorResolver,
): void => {
  // Resolve every element's colors before mutating anything: a live getComputedStyle
  // resolver depends on the <style> block and ancestor paint that the rewrite removes.
  const resolved = collectVisualElements(svg).map((el): [SVGElement, SymbolColors] => [
    el,
    resolveColors(el),
  ]);
  resolved.forEach(([el, { fill, stroke }]) => {
    setPaint(el, "fill", fill);
    setPaint(el, "stroke", stroke);
  });
  svg.querySelectorAll("style").forEach((styleEl) => styleEl.remove());
  Array.from(svg.children).forEach(addRegionIDs);
};

const paintToAttribute = (computed: string): string | null => {
  const trimmed = computed.trim();
  if (trimmed.length === 0) return null;
  // Paint servers (gradients, patterns) are not reducible to a single color; preserve
  // the reference verbatim.
  if (trimmed.startsWith("url(")) return trimmed;
  if (trimmed === "none") return "none";
  const parsed = color.fromCSS(trimmed);
  // Fall back to the raw computed value for anything unparseable (e.g. context-fill);
  // it remains a valid presentation-attribute value.
  return parsed == null ? trimmed : color.hex(parsed);
};

const computedColorResolver: ColorResolver = (el) => {
  const computed = getComputedStyle(el);
  return {
    fill: paintToAttribute(computed.fill),
    stroke: paintToAttribute(computed.stroke),
  };
};

/// normalizeSVG parses an SVG string, rewrites it into the canonical symbol form (see
/// normalizeElement), and serializes it back. Colors are resolved with the browser's
/// cascade engine, so it must run in a DOM environment; the SVG is briefly attached
/// off-screen so getComputedStyle can resolve <style> rules and inherited paint. Throws
/// a ValidationError if the input is empty or does not parse as an SVG, so callers can
/// surface the failure to the user rather than storing an unusable symbol.
export const normalizeSVG = (svgString: string): string => {
  if (svgString.trim().length === 0) throw new ValidationError("The file is empty.");
  const doc = new DOMParser().parseFromString(svgString, "image/svg+xml");
  const root = doc.documentElement;
  // A malformed document surfaces either as a non-<svg> root (the whole document is
  // replaced) or as an embedded <parsererror> node, depending on the browser.
  if (root.tagName !== "svg" || doc.querySelector("parsererror") != null)
    throw new ValidationError("The file does not contain a valid SVG image.");
  const svg = root as unknown as SVGSVGElement;
  const host = document.createElement("div");
  host.setAttribute("aria-hidden", "true");
  host.style.cssText =
    "position:absolute;width:0;height:0;overflow:hidden;opacity:0;pointer-events:none";
  host.appendChild(svg);
  document.body.appendChild(host);
  try {
    normalizeElement(svg, computedColorResolver);
  } finally {
    host.remove();
  }
  return new XMLSerializer().serializeToString(svg);
};
