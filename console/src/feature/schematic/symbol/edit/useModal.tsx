// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Divider,
  Flex,
  Form,
  Header,
  Icon,
  Input,
  Nav,
  Schematic,
  Theming,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { type bounds, color, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { HandleList } from "@/feature/schematic/symbol/edit/HandleList";
import { Preview } from "@/feature/schematic/symbol/edit/Preview";
import { RegionList } from "@/feature/schematic/symbol/edit/RegionList";
import { StateList } from "@/feature/schematic/symbol/edit/StateList";
import { SelectVariantField } from "@/feature/schematic/symbol/SelectVariant";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface ModalParams {
  /** When provided, the modal edits the existing symbol with this key. */
  symbolKey?: Schematic.Symbol.FormQuery["key"];
  parent?: ontology.ID;
  scale?: number;
  /// createKey, when set on a create-mode open, forces the new symbol to be
  /// persisted under this key. Used by the missing-symbol re-link flow so
  /// that creating a replacement symbol heals every node that already
  /// references the dangling key, without a manual re-link.
  createKey?: string;
}

const CREATE_NAME = "Schematic.Create Symbol";
const EDIT_SYMBOL_NAME = "Schematic.Edit Symbol";

const SCALE_BOUNDS: bounds.Bounds = { lower: 5, upper: 1000 };

const DEFAULT_REGION_KEY = "default";

export const useModal = Modals.create<ModalParams>(
  ({ symbolKey, parent, createKey, close }): ReactElement => {
    const isCreate = symbolKey == null;
    const theme = Theming.use();
    const { form, save } = Schematic.Symbol.useForm({
      query: { key: symbolKey },
      initialValues: {
        version: 1,
        key: createKey,
        name: "",
        parent: parent ?? ontology.ROOT_ID,
        data: {
          svg: "",
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
          handles: [],
          variant: "static",
          scaleStroke: false,
          states: [{ key: "base", name: "Base", regions: [] }],
          scale: 1,
        },
      },
      afterSave: () => close(),
    });
    const [selectedState, setSelectedState, selectedStateRef] =
      useCombinedStateAndRef<string>("base");
    const [selectedRegion, setSelectedRegion, selectedRegionRef] =
      useCombinedStateAndRef<string | undefined>(DEFAULT_REGION_KEY);
    const [selectedHandle, setSelectedHandle] = useState<string | undefined>(undefined);

    const addNewRegion = () => {
      const currentState = form.get<schematic.symbol.State>(
        `data.states.${selectedStateRef.current}`,
      ).value;
      const newRegion: schematic.symbol.Region = {
        key: `reg-${id.create()}`,
        name: `Region ${currentState.regions.length + 1}`,
        selectors: [],
        strokeColor: color.hex(theme.colors.gray.l10),
        fillColor: color.hex(color.setAlpha(theme.colors.gray.l10, 0)),
      };

      form.set(`data.states.${selectedStateRef.current}.regions`, [
        ...currentState.regions,
        newRegion,
      ]);
      setSelectedRegion(newRegion.key);
    };

    const addNewHandle = () => {
      const currentHandles = form.get<schematic.symbol.Handle[]>("data.handles").value;
      const newHandle: schematic.symbol.Handle = {
        key: `handle-${id.create()}`,
        position: { x: 0.5, y: 0.5 },
        orientation: "left",
      };
      form.set("data.handles", [...currentHandles, newHandle]);
      setSelectedHandle(newHandle.key);
    };

    const handleHandlePlace = (handleKey: string, position: xy.XY) => {
      const currentHandles = form.get<schematic.symbol.Handle[]>("data.handles").value;
      const handleIndex = currentHandles.findIndex((h) => h.key === handleKey);

      if (handleIndex !== -1) {
        const updatedHandles = [...currentHandles];
        updatedHandles[handleIndex] = { ...updatedHandles[handleIndex], position };
        form.set("data.handles", updatedHandles);
      }
    };

    const handleElementClick = (selector: string) => {
      const currentState = selectedStateRef.current;
      const currentRegion = selectedRegionRef.current;
      const regionPath = `data.states.${currentState}.regions.${currentRegion}`;
      const region = form.get<schematic.symbol.Region>(regionPath).value;
      const hasSelector = region.selectors.includes(selector);
      if (hasSelector) {
        const updatedSelectors = region.selectors.filter((s) => s !== selector);
        form.set(regionPath, { ...region, selectors: updatedSelectors });
      } else {
        const allRegions = form.get<schematic.symbol.Region[]>(
          `data.states.${currentState}.regions`,
        ).value;
        allRegions.forEach((r, index) => {
          if (!r.selectors.includes(selector)) return;
          const updatedRegion = {
            ...r,
            selectors: r.selectors.filter((s) => s !== selector),
          };
          form.set(`data.states.${currentState}.regions.${index}`, updatedRegion);
        });
        const updatedSelectors = [...region.selectors, selector];
        form.set(regionPath, { ...region, selectors: updatedSelectors });
      }
    };
    const hasSVG =
      Form.useFieldValue<string, string, typeof Schematic.Symbol.formSchema>(
        "data.svg",
        {
          ctx: form,
        },
      ).length > 0;
    const createSaveText = isCreate ? "Create" : "Save";

    return (
      <Form.Form<typeof Schematic.Symbol.formSchema> {...form}>
        <Modals.Frame
          className={CSS.BE("schematic", "symbol-create-layout")}
          background={1}
        >
          <Modals.Header icon={<Icon.Schematic />}>
            {isCreate ? CREATE_NAME : EDIT_SYMBOL_NAME}
          </Modals.Header>
          <Modals.Body full>
            <Flex.Box x grow>
              {hasSVG && (
                <Flex.Box
                  className={CSS.BE("schematic", "sidebar")}
                  y
                  rounded={1}
                  background={0}
                  bordered
                  borderColor={6}
                >
                  <Flex.Box className={CSS.BE("schematic", "name")}>
                    <Form.TextField
                      path="name"
                      inputProps={{
                        placeholder: "Symbol Name",
                        variant: "text",
                        level: "h4",
                      }}
                    />
                  </Flex.Box>
                  <Divider.Divider x />
                  <Flex.Box y>
                    <Header.Header level="p" bordered={false}>
                      <Header.Title level="p" weight={500}>
                        States
                      </Header.Title>
                    </Header.Header>
                    <Flex.Box className={CSS.BE("schematic", "states")}>
                      <SelectVariantField onSelectState={setSelectedState} />
                      <StateList value={selectedState} onChange={setSelectedState} />
                    </Flex.Box>
                  </Flex.Box>
                  <Divider.Divider x />
                  <RegionList
                    value={selectedRegion}
                    onChange={(value) => {
                      setSelectedRegion(value);
                      setSelectedHandle(undefined);
                    }}
                    selectedState={selectedState}
                    onAddRegion={addNewRegion}
                  />
                  <Divider.Divider x />
                  <HandleList
                    value={selectedHandle}
                    onChange={setSelectedHandle}
                    onAddHandle={addNewHandle}
                  />
                  <Divider.Divider x />
                  <Flex.Box y>
                    <Header.Header level="p" bordered={false}>
                      <Header.Title level="p" weight={500}>
                        Properties
                      </Header.Title>
                    </Header.Header>
                    <Flex.Box className={CSS.BE("schematic", "properties")}>
                      <Form.Field<number>
                        path="data.scale"
                        helpText="Sets the default scale when added to a schematic"
                      >
                        {({ onChange, value }) => (
                          <Input.Numeric
                            value={Math.round(value * 100)}
                            onChange={(v) => onChange(v / 100)}
                            bounds={SCALE_BOUNDS}
                            dragScale={0.5}
                            endContent="%"
                          />
                        )}
                      </Form.Field>
                      <Form.SwitchField
                        path="data.scaleStroke"
                        label="Scale stroke"
                        align="start"
                      />
                    </Flex.Box>
                  </Flex.Box>
                </Flex.Box>
              )}
              <Form.Field<string>
                path="data.svg"
                showLabel={false}
                showHelpText={false}
              >
                {({ onChange }) => (
                  <Preview
                    selectedState={selectedState}
                    selectedHandle={selectedHandle}
                    onElementClick={handleElementClick}
                    onContentsChange={onChange}
                    onHandlePlace={handleHandlePlace}
                    onHandleSelect={setSelectedHandle}
                  />
                )}
              </Form.Field>
            </Flex.Box>
          </Modals.Body>
          {hasSVG && (
            <Modals.Footer background={0}>
              <Triggers.SaveHelpText action={createSaveText} />
              <Nav.Bar.End>
                <Button.Button variant="filled" onClick={() => save()}>
                  {createSaveText}
                </Button.Button>
              </Nav.Bar.End>
            </Modals.Footer>
          )}
        </Modals.Frame>
      </Form.Form>
    );
  },
);
