import "@/schematic/symbols/Create.css";

import { ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Form,
  type Input,
  List,
  Select,
  Symbol,
  Tag,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { FileDrop } from "@/schematic/symbols/FileDrop";

export const CREATE_LAYOUT_TYPE = "schematic_edit_symbol";

export interface CreateLayoutArgs extends Symbol.UseFormParams {}

export const CREATE_LAYOUT: Layout.BaseState = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Schematic.Create Symbol",
  icon: "Schematic",
  window: {
    resizable: false,
    size: { width: 1200, height: 800 },
    navTop: true,
    showTitle: true,
  },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({
  ...CREATE_LAYOUT,
  args: initial,
});

interface SelectionState {
  selectedState: string | null;
  selectedRegion: string | null;
}

interface StateListProps extends Input.Control<string> {}

const StateListItem = (props: List.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const state = Form.useFieldValue<schematic.symbol.State>(`data.states.${itemKey}`);
  const { remove } = Form.useFieldListUtils<string, schematic.symbol.State>(
    `data.states`,
  );
  const { selected, onSelect } = Select.useItemState(itemKey);
  if (state == null) return null;
  return (
    <Button.Button
      variant={selected ? "filled" : "outlined"}
      justify="center"
      onClick={onSelect}
      style={{ minWidth: 80 }}
    >
      {state.name}
    </Button.Button>
  );
};

const stateListItem = Component.renderProp(StateListItem);

const StateList = ({ value, onChange }: StateListProps) => {
  const { data } = Form.useFieldList<string, schematic.symbol.State>("data.states");
  return (
    <Select.Frame
      value={value}
      onChange={onChange}
      data={data}
      closeDialogOnSelect={false}
    >
      <List.Items x gap={1}>
        {stateListItem}
      </List.Items>
    </Select.Frame>
  );
};

export interface RegionListProps extends Input.Control<string> {
  selectedState: string;
}

export const RegionListItem = (props: List.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const region = Form.useFieldValue<schematic.symbol.Region>(
    `data.states.${selectedState}.regions.${itemKey}`,
  );
  return <Tag.Tag>{region.name}</Tag.Tag>;
};

export const regionListItem = Component.renderProp(RegionListItem);

const RegionList = ({ value, onChange, selectedState }: RegionListProps) => {
  const { data } = Form.useFieldList<string, schematic.symbol.Region>(
    `data.states.${selectedState}.regions`,
  );
  return (
    <Select.Frame
      value={value}
      onChange={onChange}
      data={data}
      closeDialogOnSelect={false}
    >
      <List.Items gap={1} style={{ width: 200 }}>
        {regionListItem}
      </List.Items>
    </Select.Frame>
  );
};

const Preview = ({ svg }: { svg: string }): ReactElement | null => {
  const svgContainerRef = useRef<HTMLDivElement>(null);

  const injectSVG = (svgString: string) => {
    if (svgContainerRef.current == null) return;
    const parser = new DOMParser();
    const svgDoc = parser.parseFromString(svg, "image/svg+xml");
    const svgElement = svgDoc.documentElement;
    svgContainerRef.current.innerHTML = "";

    const addInteractivity = (el: Element) => {
      if (!(el instanceof SVGElement) || el.tagName === "svg") return;
      el.addEventListener("mouseenter", () => {
        if (el.tagName === "line" || el.tagName === "polyline")
          el.style.filter = "drop-shadow(0 0 3px rgba(99, 102, 241, 0.8))";
        else {
          const fillAttr = el.getAttribute("fill");
          const hasVisibleFill =
            fillAttr && fillAttr !== "none" && fillAttr !== "transparent";
          if (hasVisibleFill) el.style.filter = "brightness(1.3)";
          else {
            el.style.fill = "rgba(99, 102, 241, 0.2)";
            el.style.fillOpacity = "1";
          }
          el.classList.add(CSS.BEM("schematic", "svg-region", "hover"));
        }
      });
    };

    svgContainerRef.current.appendChild(svgElement);
  };

  useEffect(() => {}, [svg]);
  return <div ref={svgContainerRef} className={CSS.B("preview")}></div>;
};

export const Create: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const params = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const baseKey = id.create();
  const { form } = Symbol.useForm({
    params,
    initialValues: {
      name: "New Symbol",
      parent: ontology.ROOT_ID,
      data: {
        svg: "",
        states: [
          { key: baseKey, name: "Base", regions: [], color: "#000000" },
          { key: id.create(), name: "Active", regions: [], color: "#000000" },
        ],
      },
    },
  });
  const [selectedState, setSelectedState] = useState<string>(baseKey);
  const [selectedRegion, setSelectedRegion] = useState<string>("");
  return (
    <Form.Form<typeof Symbol.formSchema> {...form}>
      <Flex.Box className={CSS.BE("schematic", "symbol-create-layout")} empty full x>
        <Flex.Box className="console-form" grow full y>
          <Flex.Box x>
            <StateList value={selectedState} onChange={setSelectedState} />
          </Flex.Box>
          <Flex.Box x full>
            <RegionList
              value={selectedRegion}
              onChange={setSelectedRegion}
              selectedState={selectedState}
            />
            <Form.Field<string> path="data.svg" showLabel={false} showHelpText={false}>
              {({ onChange, value }) => (
                <FileDrop onContentsChange={onChange} grow={0}>
                  {value.length > 0 ? <Preview svg={value} /> : null}
                </FileDrop>
              )}
            </Form.Field>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
