// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";

// Visual SVG elements that should be considered for region extraction
const VISUAL_ELEMENTS = [
  "path",
  "rect",
  "circle",
  "ellipse",
  "polygon",
  "polyline",
  "line",
  "text",
];

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

export const extractRegions = (svgElement: SVGElement): schematic.symbol.Region[] => {
  const visualElements = VISUAL_ELEMENTS.flatMap((selector) =>
    Array.from(svgElement.querySelectorAll<SVGElement>(selector)),
  );
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
