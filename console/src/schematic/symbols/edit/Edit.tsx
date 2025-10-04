// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/symbols/edit/Edit.css";

import { ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Divider,
  Flex,
  Form,
  Header,
  Input,
  Nav,
  Schematic,
  Theming,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { type bounds, color, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { HandleList } from "@/schematic/symbols/edit/HandleList";
import { Preview } from "@/schematic/symbols/edit/Preview";
import { RegionList } from "@/schematic/symbols/edit/RegionList";
import { StateList } from "@/schematic/symbols/edit/StateList";
import { SelectVariantField } from "@/schematic/symbols/SelectVariant";
import { Triggers } from "@/triggers";

export const EDIT_LAYOUT_TYPE = "schematic_edit_symbol";

export interface CreateLayoutArgs extends Schematic.Symbol.FormQuery {
  parent?: ontology.ID;
  scale?: number;
}

const CREATE_NAME = "Schematic.Create Symbol";
const EDIT_NAME = "Schematic.Edit Symbol";

const EDIT_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  location: "modal",
  name: CREATE_NAME,
  icon: "Schematic",
  args: {},
  window: {
    resizable: false,
    size: { width: 1200, height: 800 },
    navTop: true,
    showTitle: true,
  },
};

export const createEditLayout = (
  initial: Partial<Layout.BaseState<CreateLayoutArgs>> = {},
): Layout.BaseState<CreateLayoutArgs> => {
  const isEdit = initial.args?.key != null;
  return { ...EDIT_LAYOUT, ...initial, name: isEdit ? EDIT_NAME : CREATE_NAME };
};

const SCALE_BOUNDS: bounds.Bounds = { lower: 5, upper: 1001 };

const DEFAULT_REGION_KEY = "default";

export const Edit: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const { key, parent } = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const isCreate = key == null;
  const dispatch = useDispatch();
  const handleUnsavedChanges = useCallback(
    (unsavedChanges: boolean) => {
      dispatch(Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges }));
    },
    [dispatch, layoutKey],
  );

  const theme = Theming.use();
  const { form, save } = Schematic.Symbol.useForm({
    query: { key },
    onHasTouched: handleUnsavedChanges,
    initialValues: {
      version: 1,
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
    afterSave: () => {
      dispatch(Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges: false }));
      onClose();
    },
  });
  const [selectedState, setSelectedState, selectedStateRef] =
    useCombinedStateAndRef<string>("base");
  const [selectedRegion, setSelectedRegion, selectedRegionRef] = useCombinedStateAndRef<
    string | undefined
  >(DEFAULT_REGION_KEY);
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
    Form.useFieldValue<string, string, typeof Schematic.Symbol.formSchema>("data.svg", {
      ctx: form,
    }).length > 0;
  const createSaveText = isCreate ? "Create" : "Save";

  return (
    <Form.Form<typeof Schematic.Symbol.formSchema> {...form}>
      <Flex.Box
        className={CSS.BE("schematic", "symbol-create-layout")}
        empty
        full
        background={1}
        y
      >
        <Flex.Box className="console-form" grow full y>
          <Flex.Box x grow>
            {hasSVG && (
              <Flex.Box
                y
                rounded={1}
                background={0}
                bordered
                borderColor={5}
                style={{
                  boxShadow: "var(--pluto-shadow-v2)",
                  minWidth: 300,
                  maxWidth: 300,
                  overflowY: "scroll",
                }}
              >
                <Flex.Box style={{ padding: "2rem 2rem 0 2rem" }}>
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
                  <Header.Header level="p" bordered={false} padded>
                    <Header.Title level="p" weight={500}>
                      States
                    </Header.Title>
                  </Header.Header>
                  <Flex.Box style={{ padding: "0 2rem" }}>
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
                  <Header.Header level="p" bordered={false} padded>
                    <Header.Title level="p" weight={500}>
                      Properties
                    </Header.Title>
                  </Header.Header>
                  <Flex.Box style={{ padding: "0 2rem" }}>
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
                      label="Scale Stroke"
                      align="start"
                    />
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            )}
            <Form.Field<string> path="data.svg" showLabel={false} showHelpText={false}>
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
        </Flex.Box>
        {hasSVG && (
          <Modals.BottomNavBar background={0}>
            <Triggers.SaveHelpText action={createSaveText} />
            <Nav.Bar.End>
              <Button.Button variant="filled" onClick={() => save()}>
                {createSaveText}
              </Button.Button>
            </Nav.Bar.End>
          </Modals.BottomNavBar>
        )}
      </Flex.Box>
    </Form.Form>
  );
};
