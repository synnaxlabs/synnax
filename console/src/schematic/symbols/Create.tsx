import "@/schematic/symbols/Create.css";

import { ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Divider,
  Flex,
  Form,
  Header,
  Input,
  Nav,
  SchematicSymbol,
  Theming,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { type bounds, color, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { HandleList } from "@/schematic/symbols/HandleList";
import { Preview } from "@/schematic/symbols/Preview";
import { RegionList } from "@/schematic/symbols/RegionList";
import { SelectVariantField } from "@/schematic/symbols/SelectVariant";
import { StateList } from "@/schematic/symbols/StateList";
import { Triggers } from "@/triggers";

export const CREATE_LAYOUT_TYPE = "schematic_edit_symbol";

export interface CreateLayoutArgs extends SchematicSymbol.UseFormParams {
  scale?: number;
}

const CREATE_NAME = "Schematic.Create Symbol";
const EDIT_NAME = "Schematic.Edit Symbol";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
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

export const createCreateLayout = (
  initial: Partial<Layout.BaseState<CreateLayoutArgs>> = {},
): Layout.BaseState<CreateLayoutArgs> => {
  const isEdit = initial.args?.key != null;
  return { ...CREATE_LAYOUT, ...initial, name: isEdit ? EDIT_NAME : CREATE_NAME };
};

const SCALE_BOUNDS: bounds.Bounds = { lower: 10, upper: 500 };

export const Create: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const params = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const isEdit = params.key != null;
  const baseRegionID = `base-region-${id.create()}`;
  const dispatch = useDispatch();
  const handleUnsavedChanges = useCallback(
    (hasUnsavedChanges: boolean) => {
      dispatch(
        Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges: hasUnsavedChanges }),
      );
    },
    [dispatch, layoutKey],
  );

  const theme = Theming.use();
  const { form, save } = SchematicSymbol.useForm({
    params,
    onHasTouched: handleUnsavedChanges,
    initialValues: {
      name: "",
      parent: ontology.ROOT_ID,
      data: {
        svg: "",
        handles: [],
        variant: "static",
        states: [
          {
            key: "base",
            name: "Base",
            regions: [
              {
                key: baseRegionID,
                name: "All Elements",
                selectors: [],
                strokeColor: color.hex(theme.colors.gray.l10),
                fillColor: color.hex(color.setAlpha(theme.colors.gray.l10, 0)),
              },
            ],
            color: "#000000",
          },
        ],
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
  >(baseRegionID);
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
    const regionPath = `data.states.${selectedStateRef.current}.regions.${selectedRegionRef.current}`;
    const region = form.get<schematic.symbol.Region>(regionPath).value;
    const hasSelector = region.selectors.includes(selector);
    const updatedSelectors = hasSelector
      ? region.selectors.filter((s) => s !== selector)
      : [...region.selectors, selector];
    form.set(regionPath, { ...region, selectors: updatedSelectors });
  };
  const hasSVG =
    Form.useFieldValue<string, string, typeof SchematicSymbol.formSchema>("data.svg", {
      ctx: form,
    }).length > 0;

  return (
    <Form.Form<typeof SchematicSymbol.formSchema> {...form}>
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
                    <SelectVariantField />
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
                          endContent={"%"}
                        />
                      )}
                    </Form.Field>
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            )}
            <Form.Field<string> path="data.svg" showLabel={false} showHelpText={false}>
              {({ onChange }) => (
                <Preview
                  selectedState={selectedState}
                  selectedRegion={selectedRegion}
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
            <Triggers.SaveHelpText action="Save to Synnax" />
            <Nav.Bar.End>
              <Button.Button variant="filled" onClick={() => save()}>
                {isEdit ? "Save" : "Create"}
              </Button.Button>
            </Nav.Bar.End>
          </Modals.BottomNavBar>
        )}
      </Flex.Box>
    </Form.Form>
  );
};
