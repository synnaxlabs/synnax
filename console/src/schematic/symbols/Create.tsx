import "@/schematic/symbols/Create.css";

import { ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Component,
  Flex,
  Form,
  Icon,
  type Input,
  List,
  Nav,
  Select,
  Symbol,
  Text,
  useCombinedRefs,
  useCombinedStateAndRef,
  useSize,
} from "@synnaxlabs/pluto";
import { box, color, id, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { FileDrop } from "@/schematic/symbols/FileDrop";
import { HandleOverlay } from "@/schematic/symbols/Handles";
import { SelectVariantField } from "@/schematic/symbols/SelectVariant";
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

export interface RegionListProps extends Input.Control<string | undefined> {
  selectedState: string;
  onAddRegion: () => void;
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
              <Text.Editable
                value={value}
                onChange={onChange}
                style={{ minWidth: 80 }}
              />
            )}
          </Form.Field>
        </Flex.Box>
        <Flex.Box x align="center" gap={1}>
          <Text.Text level="small" color={7}>
            ({region?.selectors?.length || 0} Elements)
          </Text.Text>
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

const RegionList = ({
  value,
  onChange,
  selectedState,
  onAddRegion,
}: RegionListProps) => {
  const { data } = Form.useFieldList<string, schematic.symbol.Region>(
    `data.states.${selectedState}.regions`,
  );
  return (
    <Flex.Box y gap={1} style={{ width: 300 }}>
      <Flex.Box x align="center" justify="between" style={{ padding: "1rem 2rem" }}>
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

interface HandleListProps extends Input.Control<string | undefined> {
  onAddHandle: () => void;
}

interface HandleListItemProps extends List.ItemRenderProps<string> {}

const HandleListItem = (props: HandleListItemProps) => {
  const { itemKey, index } = props;
  const path = `data.handles.${itemKey}`;
  const handle = Form.useFieldValue<{
    key: string;
    position: { x: number; y: number };
  }>(path);
  const { remove } = Form.useFieldListUtils<
    string,
    { key: string; position: { x: number; y: number } }
  >("data.handles");
  if (handle == null) return null;
  const scaledPos = xy.scale(handle.position, 100);
  return (
    <Select.ListItem {...props}>
      <Flex.Box x align="center" gap={2} justify="between" style={{ width: "100%" }}>
        <Flex.Box x align="center" gap={1}>
          <Text.Text level="small" weight={500}>
            Handle {index + 1}
          </Text.Text>
          <Text.Text level="small" color={7}>
            ({Math.round(scaledPos.x)}%, {Math.round(scaledPos.y)}%)
          </Text.Text>
        </Flex.Box>
        <Button.Button onClick={() => remove(itemKey)} size="small" variant="text">
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </Select.ListItem>
  );
};

const HandleList = ({ value, onChange, onAddHandle }: HandleListProps) => {
  const { data } = Form.useFieldList<
    string,
    { key: string; position: { x: number; y: number } }
  >("data.handles");

  return (
    <Flex.Box y gap={1} style={{ width: 300 }}>
      <Flex.Box x align="center" justify="between" style={{ padding: "1rem 2rem" }}>
        <Text.Text level="p" weight={500}>
          Handles
        </Text.Text>
        <Button.Button onClick={onAddHandle} size="small" variant="outlined">
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
          {({ key, index }) => <HandleListItem key={key} itemKey={key} index={index} />}
        </List.Items>
      </Select.Frame>
    </Flex.Box>
  );
};

interface PreviewProps {
  selectedState: string;
  selectedRegion?: string;
  selectedHandle?: string;
  onElementClick: (selector: string) => void;
  onContentsChange: (contents: string) => void;
  onHandleSelect: (handleKey: string) => void;
  onHandlePlace: (handleKey: string, position: { x: number; y: number }) => void;
}

const preprocessSVG = (svgString: string): string => {
  const parser = new DOMParser();
  const svgDoc = parser.parseFromString(svgString, "image/svg+xml");
  const svgElement = svgDoc.documentElement;
  const addRegionIds = (el: Element) => {
    if (!(el instanceof SVGElement) || el.tagName === "svg") return;
    if (!el.id && !el.getAttribute("data-region-id"))
      el.setAttribute("data-region-id", `region-${id.create()}`);
    Array.from(el.children).forEach(addRegionIds);
  };
  Array.from(svgElement.children).forEach(addRegionIds);
  const serializer = new XMLSerializer();
  return serializer.serializeToString(svgElement);
};

const Preview = ({
  selectedState,
  selectedRegion,
  selectedHandle,
  onElementClick,
  onContentsChange,
  onHandlePlace,
  onHandleSelect,
}: PreviewProps): ReactElement | null => {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgElementRef = useRef<SVGSVGElement>(null);
  const svgWrapperRef = useRef<HTMLDivElement>(null);
  const spec = Form.useFieldValue<schematic.symbol.Spec>("data");
  const [containerSizeRef, containerSize] = useSize();
  const combinedContainerRef = useCombinedRefs(containerRef, containerSizeRef);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const fitToContainer = () => {
    if (!svgElementRef.current || !containerRef.current) return;

    const svgViewBox = svgElementRef.current.viewBox.baseVal;
    let svgWidth = svgViewBox.width || svgElementRef.current.width.baseVal.value;
    let svgHeight = svgViewBox.height || svgElementRef.current.height.baseVal.value;

    if (!svgWidth || !svgHeight) {
      const bbox = svgElementRef.current.getBBox();
      svgWidth = bbox.width;
      svgHeight = bbox.height;
    }

    const containerWidth = containerRef.current.clientWidth - 100;
    const containerHeight = containerRef.current.clientHeight - 100;

    const scaleX = containerWidth / svgWidth;
    const scaleY = containerHeight / svgHeight;
    const scale = Math.min(scaleX, scaleY, 1);

    setZoom(scale);
    setPan({ x: 0, y: 0 });
  };

  const handleZoomIn = () => setZoom((z) => Math.min(z * 1.2, 5));
  const handleZoomOut = () => setZoom((z) => Math.max(z / 1.2, 0.1));
  const handleResetZoom = () => fitToContainer();

  const handleWheel = (e: React.WheelEvent) => {
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      const delta = e.deltaY > 0 ? 0.9 : 1.1;
      setZoom((z) => Math.max(0.1, Math.min(5, z * delta)));
    }
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button === 1 || (e.button === 0 && e.shiftKey)) {
      e.preventDefault();
      setIsDragging(true);
      setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) setPan({ x: e.clientX - dragStart.x, y: e.clientY - dragStart.y });
  };

  const handleMouseUp = () => setIsDragging(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!spec.svg) return;

      if ((e.ctrlKey || e.metaKey) && e.key === "0") {
        e.preventDefault();
        handleResetZoom();
      } else if ((e.ctrlKey || e.metaKey) && e.key === "=") {
        e.preventDefault();
        handleZoomIn();
      } else if ((e.ctrlKey || e.metaKey) && e.key === "-") {
        e.preventDefault();
        handleZoomOut();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [spec.svg]);

  const injectSVG = (svgContainer: HTMLDivElement, svgString: string) => {
    const parser = new DOMParser();
    const svgDoc = parser.parseFromString(svgString, "image/svg+xml");
    const svgElement = svgDoc.documentElement as unknown as SVGSVGElement;
    svgElementRef.current = svgElement;
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

      el.addEventListener("click", () => {
        let selector: string;
        if (el.id) selector = `#${el.id}`;
        else {
          const existingDataId = el.getAttribute("data-region-id");
          if (existingDataId) selector = `[data-region-id="${existingDataId}"]`;
          else {
            const newDataId = `region-${id.create()}`;
            el.setAttribute("data-region-id", newDataId);
            selector = `[data-region-id="${newDataId}"]`;
          }
        }
        onElementClick(selector);
      });

      Array.from(el.children).forEach(addInteractivity);
    };

    Array.from(svgElement.children).forEach(addInteractivity);

    svgContainer.appendChild(svgElement);

    setTimeout(() => fitToContainer(), 100);
  };

  const applyLivePreview = () => {
    if (!containerRef.current || !spec) return;

    const allElements = containerRef.current.querySelectorAll("*");
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
          const element = containerRef.current?.querySelector(selector);
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
          const element = containerRef.current?.querySelector(selector);
          if (element instanceof SVGElement)
            element.classList.add(CSS.BEM("schematic", "svg-region", "active"));
        });
    }
  };

  applyLivePreview();

  const handleContentsChange = (contents: string) => {
    const processedSVG = preprocessSVG(contents);
    if (containerRef.current == null) return;
    injectSVG(containerRef.current, processedSVG);
    onContentsChange(processedSVG);
  };

  let svgBox: box.Box = box.ZERO;
  if (svgElementRef.current) svgBox = box.construct(svgElementRef.current);

  return (
    <FileDrop
      onContentsChange={handleContentsChange}
      grow={1}
      enabled={spec.svg.length == 0}
    >
      <Flex.Box
        style={{
          position: "relative",
          width: "100%",
          height: "100%",
          overflow: "hidden",
        }}
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
      >
        {spec.svg.length > 0 && (
          <Flex.Box
            style={{
              position: "absolute",
              top: 16,
              right: 16,
              zIndex: 1000,
              gap: 8,
            }}
            x
          >
            <Button.Button
              onClick={handleZoomIn}
              size="small"
              variant="outlined"
              tooltip="Zoom In"
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              onClick={handleZoomOut}
              size="small"
              variant="outlined"
              tooltip="Zoom Out"
            >
              <Icon.Subtract />
            </Button.Button>
            <Button.Button
              onClick={handleResetZoom}
              size="small"
              variant="outlined"
              tooltip="Fit to View"
            >
              <Icon.Expand />
            </Button.Button>
          </Flex.Box>
        )}
        <div
          ref={svgWrapperRef}
          style={{
            width: "100%",
            height: "100%",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
            transformOrigin: "center",
            transition: isDragging ? "none" : "transform 0.2s ease-out",
            cursor: isDragging ? "grabbing" : "default",
          }}
        >
          <div
            ref={combinedContainerRef}
            className={CSS.B("preview")}
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          />
        </div>
        <div
          style={{
            position: "absolute",
            inset: 0,
            pointerEvents: "none",
            transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
            transformOrigin: "center",
          }}
        >
          <HandleOverlay
            handles={spec.handles}
            selectedHandle={selectedHandle}
            svgBox={svgBox}
            containerBox={containerSize}
            onSelect={onHandleSelect}
            onDrag={onHandlePlace}
          />
        </div>
      </Flex.Box>
    </FileDrop>
  );
};

export const Create: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const params = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const baseRegionID = `base-region-${id.create()}`;
  const dispatch = useDispatch();
  const handleUnsavedChanges = useCallback(
    (hasUnsavedChanges: boolean) => {
      console.log("hasUnsavedChanges", hasUnsavedChanges);
      dispatch(
        Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges: hasUnsavedChanges }),
      );
    },
    [dispatch, layoutKey],
  );
  const { form, save } = Symbol.useForm({
    params,
    onHasTouched: handleUnsavedChanges,
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
                key: baseRegionID,
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
      strokeColor: color.hex("#000000"),
      fillColor: color.hex("#000000"),
    };

    form.set(`data.states.${selectedStateRef.current}.regions`, [
      ...currentState.regions,
      newRegion,
    ]);
    setSelectedRegion(newRegion.key);
  };

  const addNewHandle = () => {
    const currentHandles =
      form.get<{ key: string; position: { x: number; y: number } }[]>(
        "data.handles",
      ).value;
    const newHandle: { key: string; position: { x: number; y: number } } = {
      key: `handle-${id.create()}`,
      position: { x: 0.5, y: 0.5 },
    };
    form.set("data.handles", [...currentHandles, newHandle]);
    setSelectedHandle(newHandle.key);
  };

  const handleHandlePlace = (handleKey: string, position: { x: number; y: number }) => {
    const currentHandles =
      form.get<{ key: string; position: { x: number; y: number } }[]>(
        "data.handles",
      ).value;
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
    Form.useFieldValue<string, string, typeof Symbol.formSchema>("data.svg", {
      ctx: form,
    }).length > 0;

  return (
    <Form.Form<typeof Symbol.formSchema> {...form}>
      <Flex.Box className={CSS.BE("schematic", "symbol-create-layout")} empty full y>
        <Flex.Box className="console-form" grow full x>
          {hasSVG && (
            <Flex.Box y gap={2}>
              <Flex.Box y bordered rounded={1} background={1} grow>
                <Flex.Box y style={{ padding: "2rem" }}>
                  <SelectVariantField />
                  <StateList value={selectedState} onChange={setSelectedState} />
                </Flex.Box>
                <RegionList
                  value={selectedRegion}
                  onChange={(value) => {
                    setSelectedRegion(value);
                    setSelectedHandle(undefined);
                  }}
                  selectedState={selectedState}
                  onAddRegion={addNewRegion}
                />
              </Flex.Box>
              <Flex.Box y grow rounded={1} background={1} bordered>
                <HandleList
                  value={selectedHandle}
                  onChange={(value) => {
                    setSelectedHandle(value);
                  }}
                  onAddHandle={addNewHandle}
                />
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
