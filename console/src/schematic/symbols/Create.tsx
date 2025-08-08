import "@/schematic/symbols/Create.css";

import { ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Component,
  Flex,
  Form,
  Icon,
  Input,
  List,
  Nav,
  Select,
  Symbol,
  Text,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { color, id } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { FileDrop } from "@/schematic/symbols/FileDrop";
import { Triggers } from "@/triggers";

export const CREATE_LAYOUT_TYPE = "schematic_edit_symbol";

export interface CreateLayoutArgs extends Symbol.UseFormParams {}

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Schematic.Create Symbol",
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
): Layout.BaseState<CreateLayoutArgs> => ({ ...CREATE_LAYOUT, ...initial });

interface StateListProps extends Input.Control<string> {}

const StateListItem = (props: List.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const state = Form.useFieldValue<schematic.symbol.State>(`data.states.${itemKey}`);
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

export interface RegionListItemProps extends List.ItemRenderProps<string> {
  selectedState: string;
}

export const RegionListItem = ({ selectedState, ...props }: RegionListItemProps) => {
  const { itemKey } = props;
  const path = `data.states.${selectedState}.regions.${itemKey}`;
  const region = Form.useFieldValue<schematic.symbol.Region>(path);
  const { remove } = Form.useFieldListUtils<string, schematic.symbol.Region>(
    `data.states.${selectedState}.regions`,
  );

  return (
    <Select.ListItem {...props}>
      <Flex.Box x align="center" gap={2} justify="between" style={{ width: "100%" }}>
        <Flex.Box x align="center" gap={1}>
          <Form.Field<string> path={`${path}.name`} showLabel={false}>
            {({ onChange, value }) => (
              <Input.Text
                variant="text"
                value={value}
                onChange={onChange}
                style={{ minWidth: 80 }}
              />
            )}
          </Form.Field>
          <Text.Text level="small" color={7}>
            ({region?.selectors?.length || 0})
          </Text.Text>
        </Flex.Box>
        <Flex.Box x align="center" gap={1}>
          <Form.Field<string> path={`${path}.strokeColor`} showLabel={false}>
            {({ onChange, value }) => (
              <Color.Swatch
                value={value}
                onChange={(v) => onChange(color.hex(v))}
                size="small"
              />
            )}
          </Form.Field>
          <Form.Field<string> path={`${path}.fillColor`} showLabel={false}>
            {({ onChange, value }) => (
              <Color.Swatch
                value={value}
                onChange={(v) => onChange(color.hex(v))}
                size="small"
              />
            )}
          </Form.Field>
          <Button.Button onClick={() => remove(itemKey)} size="small" variant="text">
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Select.ListItem>
  );
};

export interface SelectVariantProps extends Input.Control<string> {}

const VARIANT_DATA: Select.StaticEntry<string>[] = [
  { key: "static", name: "Static", icon: <Icon.Auto /> },
  { key: "actuator", name: "Actuator", icon: <Icon.Channel /> },
];

export const SelectVariant = ({ value, onChange }: SelectVariantProps) => (
  <Select.Static
    data={VARIANT_DATA}
    onChange={onChange}
    value={value}
    resourceName="variant"
  />
);

const RegionList = ({
  value,
  onChange,
  selectedState,
  onAddRegion,
}: RegionListProps & { onAddRegion: () => void }) => {
  const { data } = Form.useFieldList<string, schematic.symbol.Region>(
    `data.states.${selectedState}.regions`,
  );
  return (
    <Flex.Box y gap={1} style={{ width: 300 }}>
      <Flex.Box x align="center" justify="between">
        <Text.Text level="p" weight={500}>
          Regions
        </Text.Text>
        <Button.Button onClick={onAddRegion} size="small" variant="outlined">
          <Icon.Add />
        </Button.Button>
      </Flex.Box>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={data}
        closeDialogOnSelect={false}
      >
        <List.Items<string> y gap={1}>
          {({ key, ...rest }) => (
            <RegionListItem selectedState={selectedState} key={key} {...rest} />
          )}
        </List.Items>
      </Select.Frame>
    </Flex.Box>
  );
};

interface PreviewProps {
  selectedState: string;
  selectedRegion: string;
  onElementClick: (selector: string) => void;
  onContentsChange: (contents: string) => void;
}

const injectSVG = (
  svgContainer: HTMLDivElement,
  svgString: string,
  onElementClick: (selector: string) => void,
) => {
  const parser = new DOMParser();
  const svgDoc = parser.parseFromString(svgString, "image/svg+xml");
  const svgElement = svgDoc.documentElement;
  svgContainer.innerHTML = "";

  const addInteractivity = (el: Element) => {
    if (!(el instanceof SVGElement) || el.tagName === "svg") return;
    el.classList.add(CSS.BEM("schematic", "svg-region", "hoverable"));
    if (
      el.tagName === "line" ||
      el.tagName === "polyline" ||
      (el.tagName === "path" && el.getAttribute("fill") === "none")
    ) {
      const originalStrokeWidth = el.getAttribute("stroke-width") || "1";
      el.setAttribute("data-original-stroke-width", originalStrokeWidth);
      const currentWidth = parseFloat(originalStrokeWidth);
      if (currentWidth < 5) {
        el.setAttribute("stroke-width", "5");
        el.style.strokeOpacity = "0.3";
      }
    }

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
      }
      el.classList.add(CSS.BEM("schematic", "svg-region", "hover"));
    });

    el.addEventListener("mouseleave", () => {
      el.style.filter = "";
      el.style.fill = "";
      el.style.fillOpacity = "";
      el.classList.remove(CSS.BEM("schematic", "svg-region", "hover"));
    });

    el.addEventListener("click", (e) => {
      e.stopPropagation();
      let selector: string;
      if (el.id) selector = `#${el.id}`;
      else {
        let existingDataId = el.getAttribute("data-region-id");
        if (!existingDataId) {
          existingDataId = `region-${id.create()}`;
          el.setAttribute("data-region-id", existingDataId);
        }
        selector = `[data-region-id="${existingDataId}"]`;
      }
      onElementClick(selector);
    });

    Array.from(el.children).forEach(addInteractivity);
  };

  Array.from(svgElement.children).forEach(addInteractivity);
  svgContainer.appendChild(svgElement);
};

const Preview = ({
  selectedState,
  selectedRegion,
  onElementClick,
  onContentsChange,
}: PreviewProps): ReactElement | null => {
  const svgContainerRef = useRef<HTMLDivElement>(null);
  const spec = Form.useFieldValue<schematic.symbol.Spec>("data");
  const applyLivePreview = () => {
    if (!svgContainerRef.current || !spec) return;

    const allElements = svgContainerRef.current.querySelectorAll("*");
    allElements.forEach((el) => {
      if (el instanceof SVGElement) {
        el.removeAttribute("fill");
        el.removeAttribute("fill-opacity");
        el.style.stroke = "";
        el.style.strokeWidth = "";
        el.style.filter = "";
        el.classList.remove(
          CSS.BEM("schematic", "svg-region", "selected"),
          CSS.BEM("schematic", "svg-region", "hover"),
          CSS.BEM("schematic", "svg-region", "active"),
        );
      }
    });

    const currentState = spec.states.find((s) => s.key === selectedState);
    if (currentState)
      currentState.regions.forEach((region) => {
        region.selectors.forEach((selector) => {
          const element = svgContainerRef.current?.querySelector(selector);
          if (element instanceof SVGElement) {
            if (region.fillColor) {
              element.setAttribute("fill", region.fillColor);
              element.setAttribute("fill-opacity", "1");
            }
            if (region.strokeColor) {
              element.style.stroke = region.strokeColor;
              const originalWidth = element.getAttribute("data-original-stroke-width");
              element.style.strokeWidth = originalWidth ? `${originalWidth}px` : "2px";
              element.style.strokeOpacity = "1";
            }
          }
        });
      });

    if (selectedRegion) {
      const selectedRegionData = currentState?.regions.find(
        (r) => r.key === selectedRegion,
      );
      if (selectedRegionData)
        selectedRegionData.selectors.forEach((selector) => {
          const element = svgContainerRef.current?.querySelector(selector);
          if (element instanceof SVGElement)
            element.classList.add(CSS.BEM("schematic", "svg-region", "active"));
        });
    }
  };

  applyLivePreview();

  const handleContentsChange = (contents: string) => {
    console.log(svgContainerRef.current);
    if (svgContainerRef.current != null)
      injectSVG(svgContainerRef.current, contents, onElementClick);
    onContentsChange(contents);
  };

  return (
    <FileDrop
      onContentsChange={handleContentsChange}
      grow={1}
      enabled={spec.svg.length == 0}
    >
      <div
        ref={svgContainerRef}
        className={CSS.B("preview")}
        style={{ display: "block" }}
      />
    </FileDrop>
  );
};

export const Create: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const params = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const { form, save } = Symbol.useForm({
    params,
    initialValues: {
      name: "New Symbol",
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
                key: `base-region-${id.create()}`,
                name: "All Elements",
                selectors: [],
                strokeColor: color.hex("#000000"),
                fillColor: color.hex("#000000"),
              },
            ],
            color: "#000000",
          },
        ],
      },
    },
  });
  const [selectedState, setSelectedState, selectedStateRef] =
    useCombinedStateAndRef<string>("base");
  const [selectedRegion, setSelectedRegion, selectedRegionRef] =
    useCombinedStateAndRef<string>("");

  const addNewRegion = () => {
    const currentState = form.get<schematic.symbol.State>(
      `data.states.${selectedStateRef.current}`,
    ).value;
    const newRegion: schematic.symbol.Region = {
      key: `reg-${id.create()}`,
      name: `Region ${currentState.regions.length + 1}`,
      selectors: [],
      strokeColor: color.hex("#000000"),
      fillColor: color.hex("#000000"),
    };

    form.set(`data.states.${selectedStateRef.current}.regions`, [
      ...currentState.regions,
      newRegion,
    ]);
    setSelectedRegion(newRegion.key);
  };

  const handleElementClick = (selector: string) => {
    if (selectedRegionRef.current === "") return;
    const regionPath = `data.states.${selectedStateRef.current}.regions.${selectedRegionRef.current}`;
    const region = form.get<schematic.symbol.Region>(regionPath).value;
    const hasSelector = region.selectors.includes(selector);
    const updatedSelectors = hasSelector
      ? region.selectors.filter((s) => s !== selector)
      : [...region.selectors, selector];
    form.set(regionPath, { ...region, selectors: updatedSelectors });
  };

  return (
    <Form.Form<typeof Symbol.formSchema> {...form}>
      <Flex.Box className={CSS.BE("schematic", "symbol-create-layout")} empty full y>
        <Flex.Box className="console-form" grow full x>
          <Flex.Box y gap={2}>
            <Form.Field<string>
              path="data.variant"
              showLabel={false}
              onChange={(next, { get }) => {
                const prev = get("data.variant").value;
                if (prev === next) return;
                if (next === "actuator")
                  form.set("data.states", [
                    ...form.get<schematic.symbol.State[]>("data.states").value,
                    {
                      key: "active",
                      name: "Active",
                      regions: [],
                      color: "#000000",
                    },
                  ]);
                else if (next === "static")
                  form.set("data.states", [
                    ...form
                      .get<schematic.symbol.State[]>("data.states")
                      .value.filter((s) => s.key !== "active"),
                  ]);
              }}
            >
              {({ onChange, value }) => (
                <SelectVariant value={value} onChange={onChange} />
              )}
            </Form.Field>
            <StateList value={selectedState} onChange={setSelectedState} />
            {selectedRegion && (
              <Text.Text level="small" color={7}>
                Click SVG elements to {selectedRegion ? "add to/remove from" : "select"}{" "}
                region
              </Text.Text>
            )}
            <RegionList
              value={selectedRegion}
              onChange={setSelectedRegion}
              selectedState={selectedState}
              onAddRegion={addNewRegion}
            />
          </Flex.Box>
          <Form.Field<string> path="data.svg" showLabel={false} showHelpText={false}>
            {({ onChange }) => (
              <Preview
                selectedState={selectedState}
                selectedRegion={selectedRegion}
                onElementClick={handleElementClick}
                onContentsChange={onChange}
              />
            )}
          </Form.Field>
        </Flex.Box>
        <Modals.BottomNavBar>
          <Triggers.SaveHelpText action="Save to Synnax" />
          <Nav.Bar.End>
            <Button.Button variant="filled" onClick={() => save()}>
              Create
            </Button.Button>
          </Nav.Bar.End>
        </Modals.BottomNavBar>
      </Flex.Box>
    </Form.Form>
  );
};
