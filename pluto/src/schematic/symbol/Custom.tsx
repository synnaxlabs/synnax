// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import {
  caseconv,
  color,
  deep,
  dimensions,
  direction,
  type location,
} from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef, useState } from "react";

import { Button } from "@/button";
import { Color } from "@/color";
import { Flex } from "@/flex";
import { type Flux } from "@/flux";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { useRetrieveEffect } from "@/schematic/symbol/queries";
import { Select } from "@/select";
import { Text } from "@/text";

const ORIGINAL_STROKE_ATTRIBUTE = "data-original-stroke";
const ORIGINAL_FILL_ATTRIBUTE = "data-original-fill";

interface RegionControlsProps {
  path: string;
  onReset: (path: string) => void;
  getOriginalRegion: (path: string) => schematic.symbol.Region | null;
}

/**
 * Synchronizes state overrides with the symbol specification, preserving user customizations
 * while adding new states/regions and removing obsolete ones.
 */
const syncStateOverrides = (
  currentOverrides: schematic.symbol.State[],
  specStates: schematic.symbol.State[],
): schematic.symbol.State[] => {
  // If the number of states has changed, sync states first
  let syncedStates = [...currentOverrides];

  if (currentOverrides.length !== specStates.length) {
    // Create maps for efficient lookup
    const currentStateMap = new Map(
      currentOverrides.map((state) => [state.key, state]),
    );
    const specStateMap = new Map(specStates.map((state) => [state.key, state]));

    // Remove states that are no longer in the spec
    syncedStates = syncedStates.filter((state) => specStateMap.has(state.key));

    // Add states from spec that are not in current overrides
    for (const specState of specStates)
      if (!currentStateMap.has(specState.key)) syncedStates.push(deep.copy(specState));
  }

  // Now sync regions within each state
  const finalStates = [...syncedStates];

  for (let stateIndex = 0; stateIndex < finalStates.length; stateIndex++) {
    const currentState = finalStates[stateIndex];
    const specState = specStates.find((s) => s.key === currentState.key);

    if (specState && currentState.regions.length !== specState.regions.length) {
      const currentRegions = [...currentState.regions];

      // Create maps for efficient lookup
      const currentRegionMap = new Map(
        currentRegions.map((region) => [region.key, region]),
      );
      const specRegionMap = new Map(
        specState.regions.map((region) => [region.key, region]),
      );

      // Remove regions that are no longer in the spec
      const syncedRegions = currentRegions.filter((region) =>
        specRegionMap.has(region.key),
      );

      // Add regions from spec that are not in current state
      for (const specRegion of specState.regions)
        if (!currentRegionMap.has(specRegion.key))
          syncedRegions.push(deep.copy(specRegion));

      finalStates[stateIndex] = {
        ...currentState,
        regions: syncedRegions,
      };
    }
  }

  return finalStates;
};

const RegionControls = ({
  path,
  onReset,
  getOriginalRegion,
}: RegionControlsProps): ReactElement => {
  const name = Form.useFieldValue<string>(`${path}.name`);
  const region = Form.useFieldValue<schematic.symbol.Region>(path);
  const originalRegion = getOriginalRegion(path);
  const canBeReset = !deep.equal(region, originalRegion);
  return (
    <Flex.Box x align="center">
      <Text.Text
        level="small"
        color={9}
        style={{ maxWidth: 50, width: 50 }}
        overflow="ellipsis"
      >
        {caseconv.capitalize(name)}
      </Text.Text>
      <Flex.Box x align="stretch" key={path}>
        <Form.Field<string>
          path={`${path}.strokeColor`}
          showLabel={false}
          padHelpText={false}
        >
          {({ value, onChange }) => (
            <Color.Swatch value={value} onChange={(v) => onChange(color.hex(v))} />
          )}
        </Form.Field>
        <Form.Field<string>
          path={`${path}.fillColor`}
          showLabel={false}
          padHelpText={false}
        >
          {({ value, onChange }) => (
            <Color.Swatch value={value} onChange={(v) => onChange(color.hex(v))} />
          )}
        </Form.Field>
        <Button.Button
          onClick={() => onReset(path)}
          variant="text"
          size="tiny"
          style={{ opacity: canBeReset ? 1 : 0 }}
        >
          <Icon.Reset />
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};

interface RegionListProps {
  onReset: (path: string) => void;
  getOriginalRegion: (path: string) => schematic.symbol.Region | null;
  selectedState: string;
}

const RegionList = ({ selectedState, onReset, getOriginalRegion }: RegionListProps) => {
  const { data: regions } = Form.useFieldList<string, schematic.symbol.Region>(
    `stateOverrides.${selectedState}.regions`,
  );
  return (
    <Flex.Box y align="stretch">
      {regions.map((region) => (
        <RegionControls
          key={region}
          onReset={onReset}
          path={`stateOverrides.${selectedState}.regions.${region}`}
          getOriginalRegion={getOriginalRegion}
        />
      ))}
    </Flex.Box>
  );
};

export const StateOverrideControls = (): ReactElement => {
  const specKey = Form.useFieldValue<string>("specKey");
  const form = Form.useContext();
  const [originalStates, setOriginalStates] = useState<schematic.symbol.State[]>([]);
  const { data: states } = Form.useFieldList<string, schematic.symbol.State>(
    "stateOverrides",
  );
  const [selectedState, setSelectedState] = useState<string | undefined>(states?.[0]);

  useRetrieveEffect({
    query: { key: specKey },
    onChange: useCallback(
      (res: Flux.Result<schematic.symbol.Symbol>) => {
        if (res.data?.data == null) return;
        const symbolSpec = res.data.data;
        setOriginalStates(deep.copy(symbolSpec.states));
        const currentOverrides = form.get<schematic.symbol.State[]>("stateOverrides");
        if (currentOverrides.value?.length === 0) {
          form.set("stateOverrides", deep.copy(symbolSpec.states));
          setSelectedState(symbolSpec.states[0].key);
        } else {
          const syncedStates = syncStateOverrides(
            currentOverrides.value,
            symbolSpec.states,
          );
          form.set("stateOverrides", syncedStates);
        }
      },
      [form],
    ),
  });

  const resetRegion = useCallback(
    (path: string) => {
      const parsedPath = path.split(".").slice(1).join(".");
      const prev = deep.get(originalStates, parsedPath, { optional: true });
      if (prev == null) return;
      form.set(path, deep.copy(prev));
    },
    [form.set, originalStates],
  );

  const getOriginalRegion = useCallback(
    (path: string): schematic.symbol.Region | null => {
      const parsedPath = path.split(".").slice(1).join(".");
      return deep.get(originalStates, parsedPath, { optional: true }) ?? null;
    },
    [originalStates],
  );

  return (
    <Flex.Box y align="stretch">
      {states.length > 1 && (
        <Select.Buttons
          keys={states}
          value={selectedState}
          onChange={setSelectedState}
          full="x"
        >
          {states.map((state) => (
            <Select.Button key={state} itemKey={state} justify="center">
              {caseconv.capitalize(state)}
            </Select.Button>
          ))}
        </Select.Buttons>
      )}
      {selectedState != null && (
        <RegionList
          selectedState={selectedState}
          onReset={resetRegion}
          getOriginalRegion={getOriginalRegion}
        />
      )}
    </Flex.Box>
  );
};

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

    // The state already contains the overridden colors
    const { strokeColor, fillColor } = region;

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
  stateOverrides?: schematic.symbol.State[];
}

export const useCustom = ({
  container,
  orientation,
  activeState,
  externalScale,
  spec,
  onMount,
  stateOverrides,
}: UseCustomArgs) => {
  const svgElementRef = useRef<SVGSVGElement>(null);
  const baseDimsRef = useRef<dimensions.Dimensions>({ width: 0, height: 0 });

  const prevExternalScaleRef = useRef<number | undefined>(undefined);
  const prevOrientationRef = useRef<location.Outer | undefined>(undefined);
  const prevSpecDataRef = useRef<schematic.symbol.Spec | undefined>(undefined);
  const prevStateRef = useRef<schematic.symbol.State>(undefined);
  const prevStateOverridesRef = useRef<typeof stateOverrides>(undefined);

  if (spec == null || spec.svg.length === 0 || container == null) return;

  const externalScaleDiffers = prevExternalScaleRef.current !== externalScale;
  const svgDiffers = prevSpecDataRef.current?.svg !== spec?.svg;
  const orientationDiffers = prevOrientationRef.current !== orientation;
  const internalScaleDiffers = prevSpecDataRef.current?.scale !== spec?.scale;
  const scaleStrokeDiffers = prevSpecDataRef.current?.scaleStroke !== spec?.scaleStroke;
  const specDiffers = prevSpecDataRef.current !== spec;

  // Get the current state from overrides if available, otherwise from spec
  const stateIndex = activeState === "active" ? 1 : 0;
  const currState = stateOverrides?.[stateIndex] ?? spec.states[stateIndex];

  const stateDiffers = prevStateRef.current !== currState;
  const stateOverridesDiffers =
    JSON.stringify(prevStateOverridesRef.current) !== JSON.stringify(stateOverrides);
  const different =
    externalScaleDiffers ||
    svgDiffers ||
    scaleStrokeDiffers ||
    stateDiffers ||
    stateOverridesDiffers;
  if (!different) return;
  if (externalScaleDiffers) prevExternalScaleRef.current = externalScale;
  if (orientationDiffers) prevOrientationRef.current = orientation;
  if (specDiffers) prevSpecDataRef.current = deep.copy(spec);
  if (stateOverridesDiffers) prevStateOverridesRef.current = stateOverrides;
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

  if (stateDiffers || stateOverridesDiffers) {
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
