// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/custom/Overrides.css";

import { type schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { caseconv, deep } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/css";
import {
  listValues,
  type PaintContext,
  resolveRegions,
  usePaintContext,
} from "@/schematic/node/common/custom/paint";
import { Region } from "@/schematic/node/common/region";
import { Symbol } from "@/schematic/symbol";

const root = deep.path<{
  specKey: string;
  stateOverrides: schematic.symbol.StateOverride[];
}>();

type RegionPath = deep.Path<schematic.symbol.Region>;

interface RegionControlsProps {
  path: RegionPath;
  svg: string;
  original?: schematic.symbol.Region;
}

// defaultOverrides returns an override for each value that the node of spec draws, with
// the colors that the looks give each part at that value.
const defaultOverrides = (
  spec: schematic.symbol.Spec,
  ctx: Omit<PaintContext, "value">,
): schematic.symbol.StateOverride[] => {
  const [state] = spec.states;
  const entries =
    state?.type === "enumeration" && state.enumeration != null
      ? ctx.enumerations.get(state.enumeration)?.entries
      : undefined;
  return listValues(spec, entries).map(({ key, name, value }) => ({
    key,
    name,
    regions: resolveRegions(spec, { ...ctx, value }),
  }));
};

const syncStateOverrides = (
  currentOverrides: schematic.symbol.StateOverride[],
  specStates: schematic.symbol.StateOverride[],
): schematic.symbol.StateOverride[] => {
  let syncedStates = [...currentOverrides];

  if (currentOverrides.length !== specStates.length) {
    const currentStateMap = new Map(
      currentOverrides.map((state) => [state.key, state]),
    );
    const specStateMap = new Map(specStates.map((state) => [state.key, state]));

    syncedStates = syncedStates.filter((state) => specStateMap.has(state.key));

    for (const specState of specStates)
      if (!currentStateMap.has(specState.key)) syncedStates.push(deep.copy(specState));
  }

  const finalStates = [...syncedStates];

  for (let stateIndex = 0; stateIndex < finalStates.length; stateIndex++) {
    const currentState = finalStates[stateIndex];
    const specState = specStates.find((s) => s.key === currentState.key);

    if (specState && currentState.regions.length !== specState.regions.length) {
      const currentRegions = [...currentState.regions];

      const currentRegionMap = new Map(
        currentRegions.map((region) => [region.key, region]),
      );
      const specRegionMap = new Map(
        specState.regions.map((region) => [region.key, region]),
      );

      const syncedRegions = currentRegions.filter((region) =>
        specRegionMap.has(region.key),
      );

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

const RegionControls = ({ path, svg, original }: RegionControlsProps): ReactElement => {
  const name = Form.useFieldValue(path.name);
  const region = Form.useFieldValue(path);
  const { set } = Form.useContext();
  const paint = Region.paint(svg, region.selectors);
  const canBeReset = !deep.equal(region, original);
  return (
    <Flex.Box x align="center">
      <Text.Text
        level="small"
        color={9}
        className={CSS.B("schematic-region-name")}
        overflow="fade"
      >
        {caseconv.capitalize(name)}
      </Text.Text>
      <Flex.Box x align="stretch" key={String(path)}>
        <Color.Swatch
          value={region.strokeColor}
          fallback={paint.stroke}
          onChange={(c) => set(path.strokeColor, c)}
        />
        <Color.Swatch
          value={region.fillColor}
          fallback={paint.fill}
          onChange={(c) => set(path.fillColor, c)}
        />
        <Button.Button
          onClick={() => {
            if (original != null) set(path, deep.copy(original));
          }}
          variant="text"
          size="tiny"
          className={CSS.cls(CSS.B("schematic-region-reset"), CSS.visible(canBeReset))}
        >
          <Icon.Reset />
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};

interface RegionListProps {
  svg: string;
  selectedState: string;
  originalRegions?: schematic.symbol.Region[];
}

const RegionList = ({ svg, selectedState, originalRegions }: RegionListProps) => {
  const list = root.stateOverrides[selectedState].regions;
  const { data: regions } = Form.useFieldList(list);
  return (
    <Flex.Box y align="stretch">
      {regions.map((region) => (
        <RegionControls
          key={region}
          svg={svg}
          path={list[region]}
          original={originalRegions?.find(({ key }) => key === region)}
        />
      ))}
    </Flex.Box>
  );
};

interface SymbolOverridesProps {
  spec: schematic.symbol.Spec;
}

const SymbolOverrides = ({ spec }: SymbolOverridesProps): ReactElement => {
  const form = Form.useContext();
  const ctx = usePaintContext(spec);
  const defaults = useMemoDeepEqual(
    ctx == null ? undefined : defaultOverrides(spec, ctx),
  );
  const { data: states } = Form.useFieldList(root.stateOverrides);
  const [selectedState, setSelectedState] = useState<string | undefined>(states?.[0]);

  useEffect(() => {
    if (defaults == null || defaults.length === 0) return;
    const currentOverrides = form.get(root.stateOverrides).value;
    if (currentOverrides.length === 0) {
      form.set(root.stateOverrides, deep.copy(defaults));
      setSelectedState(defaults[0].key);
    } else
      form.set(root.stateOverrides, syncStateOverrides(currentOverrides, defaults));
  }, [defaults, form]);

  // The form can swap to a different symbol's overrides beneath the locally
  // held selection, so fall back to the first state when the key disappears.
  const shownState =
    selectedState != null && states.includes(selectedState) ? selectedState : states[0];

  return (
    <Flex.Box y align="stretch">
      {states.length > 1 && (
        <Select.Buttons value={shownState} onChange={setSelectedState} full="x">
          {states.map((state) => (
            <Select.Item key={state} itemKey={state} justify="center">
              {caseconv.capitalize(state)}
            </Select.Item>
          ))}
        </Select.Buttons>
      )}
      {shownState != null && (
        <RegionList
          svg={spec.drawing.markup}
          selectedState={shownState}
          originalRegions={defaults?.find(({ key }) => key === shownState)?.regions}
        />
      )}
    </Flex.Box>
  );
};

/**
 * Edits the state overrides of a custom symbol node: for each entry of the symbol's
 * enumeration, the colors of each part.
 */
export const StateOverrideForm = (): ReactElement | null => {
  const specKey = Form.useFieldValue(root.specKey);
  // A dangling spec key must not throw here: the owner renders a repair form for it.
  const { symbol } = Symbol.useResolved(specKey);
  if (symbol == null) return null;
  return <SymbolOverrides spec={symbol.data} />;
};
