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
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import { type bounds, deep, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, useRef, useState } from "react";

import { HandleList } from "@/feature/schematic/symbol/edit/HandleList";
import { Inspector } from "@/feature/schematic/symbol/edit/Inspector";
import { PartList } from "@/feature/schematic/symbol/edit/PartList";
import { Preview } from "@/feature/schematic/symbol/edit/Preview";
import { useConfirmSave } from "@/feature/schematic/symbol/edit/useConfirmSave";
import { useValues } from "@/feature/schematic/symbol/edit/useValues";
import { SelectVariantField } from "@/feature/schematic/symbol/SelectVariant";
import { Spec } from "@/feature/schematic/symbol/spec";
import { FLAGS } from "@/flags";
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

interface ValueBarProps {
  value: number;
  onChange: (value: number) => void;
}

// ValueBar picks the value of the state that the canvas draws.
const ValueBar = ({ value, onChange }: ValueBarProps): ReactElement | null => {
  const { list } = useValues();
  if (list.length < 2) return null;
  return (
    <Flex.Box x align="center" gap="small" className={CSS.BE("schematic", "value-bar")}>
      <Text.Text level="small" color={9}>
        Preview
      </Text.Text>
      <Select.Buttons<number>
        value={value}
        onChange={onChange}
        size="small"
        variant="outlined"
      >
        {list.map((v) => (
          <Select.Item<number> key={v.value} itemKey={v.value} size="small">
            {v.name}
          </Select.Item>
        ))}
      </Select.Buttons>
    </Flex.Box>
  );
};

// climb returns what an Alt-click on shape picks for part, as Select parent group does:
// the group above the outermost shape of part there, or above the shape. With no group
// above, it returns shape when part holds none of it.
const climb = (
  spec: schematic.symbol.Spec,
  part: string | undefined,
  shape: Schematic.Node.Region.Named,
): Schematic.Node.Region.Named | null => {
  const drawn = { ...spec, drawing: { ...spec.drawing, markup: shape.markup } };
  const from = part == null ? undefined : Spec.outermost(drawn, part, shape.selector);
  const group = Spec.parentGroup(drawn, [from ?? shape.selector]);
  if (group == null) return from == null ? shape : null;
  return Schematic.Node.Region.name(
    shape.markup,
    group.ownerDocument.documentElement,
    group,
  );
};

const Editor = ({
  symbolKey,
  group,
  createKey,
  close,
}: Modals.ContentProps<ModalParams, void>): ReactElement => {
  const isCreate = symbolKey == null;
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
        states: [Spec.createState()],
        commands: [],
        looks: [],
        interactions: [],
        scale: 1,
      },
    },
    afterSave: () => close(),
  });
  Modals.useBusy(variant === "loading");
  const stored = Schematic.Symbol.useResult(
    symbolKey == null ? null : { key: symbolKey, includeUses: true },
  );
  const confirmSave = useConfirmSave();
  const [selectedValue, setSelectedValue] = useState(0);
  const [selectedPart, setSelectedPart, selectedPartRef] =
    useCombinedStateAndRef<Spec.Target>(undefined);
  const [selectedHandle, setSelectedHandle] = useState<string | undefined>(undefined);

  // addPart adds a part, with the shape of selector when one is given, and selects it.
  const addPart = (selector?: string) => {
    const parts = form.get(path.data.parts).value;
    const part: schematic.symbol.Part = {
      key: `part-${id.create()}`,
      name: Spec.nextPartName(parts),
      selectors: [],
    };
    const next = [...parts, part];
    const drawing = form.get(path.data.drawing).value;
    form.set(
      path.data.parts,
      selector == null
        ? next
        : Spec.toggleSelector({ drawing, parts: next }, part.key, selector),
    );
    setSelectedPart(part.key);
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

  const lastClickSelectsRef = useRef(false);
  const handleElementClick = (
    shape: Schematic.Node.Region.Named,
    { detail, altKey }: Pick<MouseEvent, "detail" | "altKey">,
  ) => {
    // A later click of the same double click would undo a click that selects a part.
    if (detail > 1 && lastClickSelectsRef.current) return;
    const part = selectedPartRef.current;
    const named = altKey ? climb(form.get(path.data).value, part, shape) : shape;
    if (named == null) return;
    if (named.markup !== form.get(path.data.drawing.markup).value)
      form.set(path.data.drawing.markup, named.markup);
    const { selector } = named;
    const spec = form.get(path.data).value;
    // A shape in a group of another part cannot join part, so the click selects it.
    const holder = Spec.groupOwner(spec, selector);
    const toggles = part != null && (holder == null || holder.key === part);
    lastClickSelectsRef.current = !toggles;
    if (toggles)
      return form.set(path.data.parts, Spec.toggleSelector(spec, part, selector));
    const owner = part == null ? Spec.owner(spec, selector) : holder;
    if (owner == null) return addPart(selector);
    setSelectedPart(owner.key);
  };
  const hasSVG =
    Form.useFieldValue(path.data.drawing.markup, {
      ctx: form,
    }).length > 0;
  const createSaveText = isCreate ? "Create" : "Save";
  const handleSave = async () => {
    // A fill motion that cannot draw stops the save and selects its part, whose
    // inspector shows why. The inspector shows the issues only with the flag on.
    if (FLAGS.symbolMotion) {
      const [part] = Spec.levelIssues(form.get(path.data).value).keys();
      if (part != null) return setSelectedPart(part);
    }
    const uses = stored.data?.uses ?? [];
    if (stored.data != null && uses.length > 0) {
      const spec = form.get(path.data).value;
      const changed = !deep.equal(
        { ...stored.data.data, previewViewport: undefined },
        { ...spec, previewViewport: undefined },
      );
      if (changed && (await confirmSave({ stored: stored.data, spec, uses })) !== true)
        return;
    }
    save();
  };

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
                  </Flex.Box>
                </Flex.Box>
                <Divider.Divider x />
                <PartList
                  value={selectedPart}
                  onChange={(value) => {
                    setSelectedPart(value);
                    setSelectedHandle(undefined);
                  }}
                  onAdd={() => addPart()}
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
            <Flex.Box y grow className={CSS.BE("schematic", "canvas")}>
              <Form.Field
                path={path.data.drawing.markup}
                showLabel={false}
                showHelpText={false}
              >
                {({ onChange }) => (
                  <Preview
                    selectedValue={selectedValue}
                    selectedHandle={selectedHandle}
                    selectedPart={selectedPart}
                    onElementClick={handleElementClick}
                    onContentsChange={onChange}
                    onHandlePlace={handleHandlePlace}
                    onHandleSelect={setSelectedHandle}
                  />
                )}
              </Form.Field>
              {hasSVG && <ValueBar value={selectedValue} onChange={setSelectedValue} />}
            </Flex.Box>
            {hasSVG && (
              <Flex.Box
                className={CSS.BE("schematic", "sidebar")}
                y
                rounded="small"
                background={0}
                bordered
                borderColor={6}
              >
                <Inspector
                  key={selectedPart ?? ""}
                  part={selectedPart}
                  preview={selectedValue}
                />
              </Flex.Box>
            )}
          </Flex.Box>
        </Modal.Body>
        {hasSVG && (
          <Modal.Footer background={0}>
            <Nav.Bar.End>
              <Button.Button
                variant="filled"
                status={
                  !isCreate && stored.variant === "loading"
                    ? "loading"
                    : status.keepVariants(variant, "loading")
                }
                disabled={!isCreate && stored.variant === "error"}
                onClick={() => void handleSave()}
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
