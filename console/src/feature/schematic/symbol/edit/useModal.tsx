// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { type group, type schematic, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { useCombinedStateAndRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Enumeration, Schematic } from "@synnaxlabs/pluto";
import { type bounds, color, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { HandleList } from "@/feature/schematic/symbol/edit/HandleList";
import { Preview } from "@/feature/schematic/symbol/edit/Preview";
import { RegionList } from "@/feature/schematic/symbol/edit/RegionList";
import {
  createState,
  enumerationKey,
  setColor,
  toggleSelector,
} from "@/feature/schematic/symbol/edit/spec";
import { StateList } from "@/feature/schematic/symbol/edit/StateList";
import { SelectVariantField } from "@/feature/schematic/symbol/SelectVariant";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = Schematic.Symbol.formPath;

export interface ModalParams {
  /** When provided, the modal edits the existing symbol with this key. */
  symbolKey?: Schematic.Symbol.FormQuery["key"];
  /** The group a new symbol joins. */
  group?: group.Key;
  scale?: number;
  /// createKey, when set on a create-mode open, forces the new symbol to be
  /// persisted under this key. Used by the missing-symbol re-link flow so
  /// that creating a replacement symbol heals every node that already
  /// references the dangling key, without a manual re-link.
  createKey?: string;
}

const SCALE_BOUNDS: bounds.Bounds = { lower: 5, upper: 1000 };

const DEFAULT_REGION_KEY = "default";

interface StatesProps {
  value: number;
  onChange: (value: number) => void;
}

// States lists the values of the state, named by the entries of its enumeration.
const States = ({ value, onChange }: StatesProps): ReactElement | null => {
  const spec = Form.useFieldValue(path.data);
  const key = enumerationKey(spec);
  const res = Enumeration.useResult(key == null ? null : { key });
  const entries = res.variant === "success" ? res.data.entries : undefined;
  return (
    <StateList
      value={value}
      onChange={onChange}
      tabs={Schematic.Node.Custom.listValues(spec, entries)}
    />
  );
};

const Editor = ({
  symbolKey,
  group,
  createKey,
  close,
}: Modals.ContentProps<ModalParams, void>): ReactElement => {
  const isCreate = symbolKey == null;
  const theme = Theming.use();
  const { form, save, variant } = Schematic.Symbol.useForm({
    query: symbolKey == null ? null : { key: symbolKey },
    initialValues: {
      key: createKey,
      name: "",
      group,
      data: {
        drawing: { type: "svg", markup: "" },
        parts: [],
        previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        handles: [],
        variant: "static",
        strokeScaled: false,
        states: [createState()],
        commands: [],
        looks: [],
        interactions: [],
        scale: 1,
      },
    },
    afterSave: () => close(),
  });
  Modals.useBusy(variant === "loading");
  const [selectedValue, setSelectedValue, selectedValueRef] =
    useCombinedStateAndRef<number>(0);
  const [selectedRegion, setSelectedRegion, selectedRegionRef] =
    useCombinedStateAndRef<string>(DEFAULT_REGION_KEY);
  const [selectedHandle, setSelectedHandle] = useState<string | undefined>(undefined);

  const addNewRegion = () => {
    const parts = form.get(path.data.parts).value;
    const part: schematic.symbol.Part = {
      key: `reg-${id.create()}`,
      name: `Region ${parts.length + 1}`,
      selectors: [],
    };
    const value = selectedValueRef.current;
    let looks = form.get(path.data.looks).value;
    looks = setColor(looks, part.key, value, "stroke", theme.colors.gray.l10);
    looks = setColor(
      looks,
      part.key,
      value,
      "fill",
      color.setOpacity(theme.colors.gray.l10, 0),
    );
    form.set(path.data.parts, [...parts, part]);
    form.set(path.data.looks, looks);
    setSelectedRegion(part.key);
  };

  const addNewHandle = () => {
    const currentHandles = form.get(path.data.handles).value;
    const newHandle: schematic.symbol.Handle = {
      key: `handle-${id.create()}`,
      position: { x: 0.5, y: 0.5 },
      orientation: "left",
    };
    form.set(path.data.handles, [...currentHandles, newHandle]);
    setSelectedHandle(newHandle.key);
  };

  const handleHandlePlace = (handleKey: string, position: xy.XY) => {
    const currentHandles = form.get(path.data.handles).value;
    const handleIndex = currentHandles.findIndex((h) => h.key === handleKey);

    if (handleIndex !== -1) {
      const updatedHandles = [...currentHandles];
      updatedHandles[handleIndex] = { ...updatedHandles[handleIndex], position };
      form.set(path.data.handles, updatedHandles);
    }
  };

  const handleElementClick = (selector: string) =>
    form.set(
      path.data.parts,
      toggleSelector(
        form.get(path.data.parts).value,
        selectedRegionRef.current,
        selector,
      ),
    );
  const hasSVG =
    Form.useFieldValue(path.data.drawing.markup, {
      ctx: form,
    }).length > 0;
  const createSaveText = isCreate ? "Create" : "Save";

  return (
    <Form.Form<typeof Schematic.Symbol.formSchema> {...form}>
      <Modal.Frame
        className={CSS.BE("schematic", "symbol-create-layout")}
        background={1}
      >
        <Modal.Header icon={<Icon.Schematic />}>
          {isCreate ? "Schematic.Symbol.Create" : "Schematic.Symbol.Edit"}
        </Modal.Header>
        <Modal.Body full>
          <Flex.Box x grow>
            {hasSVG && (
              <Flex.Box
                className={CSS.BE("schematic", "sidebar")}
                y
                rounded="small"
                background={0}
                bordered
                borderColor={6}
              >
                <Flex.Box className={CSS.BE("schematic", "name")}>
                  <Form.TextField
                    path={path.name}
                    inputProps={{
                      placeholder: "Symbol name",
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
                    <SelectVariantField onSelectValue={setSelectedValue} />
                    <States value={selectedValue} onChange={setSelectedValue} />
                  </Flex.Box>
                </Flex.Box>
                <Divider.Divider x />
                <RegionList
                  value={selectedRegion}
                  onChange={(value) => {
                    setSelectedRegion(value);
                    setSelectedHandle(undefined);
                  }}
                  selectedValue={selectedValue}
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
                    <Form.Field
                      path={path.data.scale}
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
                    <Form.CheckboxField
                      path={path.data.strokeScaled}
                      label="Scale stroke"
                      align="start"
                    />
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            )}
            <Form.Field
              path={path.data.drawing.markup}
              showLabel={false}
              showHelpText={false}
            >
              {({ onChange }) => (
                <Preview
                  selectedValue={selectedValue}
                  selectedHandle={selectedHandle}
                  onElementClick={handleElementClick}
                  onContentsChange={onChange}
                  onHandlePlace={handleHandlePlace}
                  onHandleSelect={setSelectedHandle}
                />
              )}
            </Form.Field>
          </Flex.Box>
        </Modal.Body>
        {hasSVG && (
          <Modal.Footer background={0}>
            <Nav.Bar.End>
              <Button.Button
                variant="filled"
                status={status.keepVariants(variant, "loading")}
                onClick={() => save()}
                trigger={Triggers.SAVE}
                triggerIndicator
              >
                {createSaveText}
              </Button.Button>
            </Nav.Bar.End>
          </Modal.Footer>
        )}
      </Modal.Frame>
    </Form.Form>
  );
};

export const useModal = Modals.create<ModalParams>(Editor);
