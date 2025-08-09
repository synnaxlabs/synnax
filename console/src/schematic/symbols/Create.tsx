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
import { type ReactElement, useRef, useState, useEffect } from "react";

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

interface HandleListProps extends Input.Control<string> {
  onAddHandle: () => void;
}

interface HandleListItemProps extends List.ItemRenderProps<string> {}

const HandleListItem = ({ itemKey }: HandleListItemProps) => {
  const path = `data.handles.${itemKey}`;
  const handle = Form.useFieldValue<{ key: string; position: { x: number; y: number } }>(path);
  const { remove } = Form.useFieldListUtils<string, { key: string; position: { x: number; y: number } }>(
    "data.handles",
  );
  const { onSelect } = Select.useItemState(itemKey);

  if (handle == null) return null;

  return (
    <Select.ListItem itemKey={itemKey} index={0} key={itemKey} onSelect={onSelect}>
      <Flex.Box x align="center" gap={2} justify="between" style={{ width: "100%" }}>
        <Flex.Box x align="center" gap={1}>
          <Text.Text level="small" weight={500}>
            Handle {itemKey.slice(-4)}
          </Text.Text>
          <Text.Text level="small" color={7}>
            ({Math.round(handle?.position?.x || 0)},{" "}
            {Math.round(handle?.position?.y || 0)})
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
  const { data } = Form.useFieldList<string, { key: string; position: { x: number; y: number } }>("data.handles");

  return (
    <Flex.Box y gap={1} style={{ width: 300 }}>
      <Flex.Box x align="center" justify="between">
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
  selectedRegion: string;
  selectedHandle: string;
  onElementClick: (selector: string) => void;
  onContentsChange: (contents: string) => void;
  onHandlePlace: (handleKey: string, position: { x: number; y: number }) => void;
}

interface SVGDimensions {
  x: number;
  y: number;
  width: number;
  height: number;
}

const parseSVGDimensions = (svgString: string): SVGDimensions => {
  const parser = new DOMParser();
  const svgDoc = parser.parseFromString(svgString, "image/svg+xml");
  const svgElement = svgDoc.documentElement;

  // Get viewBox or fallback to width/height attributes
  const viewBox = svgElement.getAttribute("viewBox");
  if (viewBox) {
    const [x, y, width, height] = viewBox.split(" ").map(Number);
    return { x, y, width, height };
  }

  // Fallback to width/height attributes
  const width = parseFloat(svgElement.getAttribute("width") || "100");
  const height = parseFloat(svgElement.getAttribute("height") || "100");
  return { x: 0, y: 0, width, height };
};

const convertToPercentage = (
  svgPos: { x: number; y: number },
  svgDimensions: SVGDimensions,
) => ({
  left: ((svgPos.x - svgDimensions.x) / svgDimensions.width) * 100,
  top: ((svgPos.y - svgDimensions.y) / svgDimensions.height) * 100,
});


const preprocessSVG = (svgString: string): string => {
  const parser = new DOMParser();
  const svgDoc = parser.parseFromString(svgString, "image/svg+xml");
  const svgElement = svgDoc.documentElement;

  const addRegionIds = (el: Element) => {
    if (!(el instanceof SVGElement) || el.tagName === "svg") return;

    // Only add data-region-id if the element doesn't have an id
    if (!el.id && !el.getAttribute("data-region-id"))
      el.setAttribute("data-region-id", `region-${id.create()}`);

    Array.from(el.children).forEach(addRegionIds);
  };

  Array.from(svgElement.children).forEach(addRegionIds);

  const serializer = new XMLSerializer();
  return serializer.serializeToString(svgElement);
};

const injectSVG = (
  svgContainer: HTMLDivElement,
  svgString: string,
  onElementClick: (selector: string) => void,
  onHandlePlace?: (position: { x: number; y: number }) => void,
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
        const existingDataId = el.getAttribute("data-region-id");
        if (existingDataId) selector = `[data-region-id="${existingDataId}"]`;
        else {
          // This shouldn't happen if preprocessSVG was called, but keep as fallback
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

  // Add handle placement click handler to the SVG container
  if (onHandlePlace)
    svgContainer.addEventListener("click", (e) => {
      // Only handle clicks on the SVG background (not on SVG elements)
      if (e.target === svgElement || e.target === svgContainer) {
        const svgRect = svgElement.getBoundingClientRect();
        const svgDimensions = parseSVGDimensions(svgString);

        const x =
          svgDimensions.x +
          ((e.clientX - svgRect.left) / svgRect.width) * svgDimensions.width;
        const y =
          svgDimensions.y +
          ((e.clientY - svgRect.top) / svgRect.height) * svgDimensions.height;

        onHandlePlace({ x, y });
      }
    });

  svgContainer.appendChild(svgElement);
};

const Preview = ({
  selectedState,
  selectedRegion,
  selectedHandle,
  onElementClick,
  onContentsChange,
  onHandlePlace,
}: PreviewProps): ReactElement | null => {
  const svgContainerRef = useRef<HTMLDivElement>(null);
  const handleOverlayRef = useRef<HTMLDivElement>(null);
  const spec = Form.useFieldValue<schematic.symbol.Spec>("data");
  const [dragState, setDragState] = useState<{
    handleKey: string | null;
    isDragging: boolean;
    startPos: { x: number; y: number };
  }>({
    handleKey: null,
    isDragging: false,
    startPos: { x: 0, y: 0 }
  });
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

  const renderHandleOverlays = () => {
    if (!spec?.svg || !spec?.handles || spec.handles.length === 0) {
      console.log("Not rendering overlays:", { 
        hasSvg: !!spec?.svg, 
        hasHandles: !!spec?.handles, 
        handleCount: spec?.handles?.length || 0 
      });
      return null;
    }
    
    const svgDimensions = parseSVGDimensions(spec.svg);
    console.log("Rendering handle overlays:", { 
      handleCount: spec.handles.length, 
      svgDimensions, 
      selectedHandle 
    });
    
    return spec.handles.map((handle) => {
      // Get the actual SVG element to calculate its real position
      const svgElement = svgContainerRef.current?.querySelector('svg');
      if (!svgElement) return null;

      const svgRect = svgElement.getBoundingClientRect();
      const containerRect = svgContainerRef.current!.getBoundingClientRect();
      
      // Calculate position within the SVG coordinate space
      const svgRelativeX = ((handle.position.x - svgDimensions.x) / svgDimensions.width);
      const svgRelativeY = ((handle.position.y - svgDimensions.y) / svgDimensions.height);
      
      // Position relative to the container, accounting for the centered SVG
      const left = ((svgRect.left - containerRect.left) / containerRect.width * 100) + 
                   (svgRelativeX * svgRect.width / containerRect.width * 100);
      const top = ((svgRect.top - containerRect.top) / containerRect.height * 100) + 
                  (svgRelativeY * svgRect.height / containerRect.height * 100);
      
      const isSelected = selectedHandle === handle.key;
      
      console.log(`Handle ${handle.key}:`, { 
        position: handle.position,
        svgDimensions,
        svgRect: {w: svgRect.width, h: svgRect.height, l: svgRect.left, t: svgRect.top},
        containerRect: {w: containerRect.width, h: containerRect.height, l: containerRect.left, t: containerRect.top},
        svgRelativeX,
        svgRelativeY,
        left,
        top,
        isSelected 
      });
      
      const isDragging = dragState.isDragging && dragState.handleKey === handle.key;

      return (
        <div
          key={handle.key}
          style={{
            position: "absolute",
            left: `${left}%`,
            top: `${top}%`,
            transform: "translate(-50%, -50%)",
            width: "12px",
            height: "12px",
            borderRadius: "50%",
            backgroundColor: isSelected ? "#6366f1" : "#6b7280",
            border: "2px solid white",
            cursor: isDragging ? "grabbing" : "grab",
            zIndex: 1000,
            boxShadow: "0 2px 4px rgba(0,0,0,0.2)",
            pointerEvents: "auto",
            opacity: isDragging ? 0.8 : 1,
            transition: isDragging ? "none" : "opacity 0.2s ease",
          }}
          onMouseDown={(e) => {
            e.preventDefault();
            e.stopPropagation();
            console.log("Starting drag for handle:", handle.key);
            setDragState({
              handleKey: handle.key,
              isDragging: true,
              startPos: { x: e.clientX, y: e.clientY }
            });
            setSelectedHandle(handle.key);
          }}
          onClick={(e) => {
            e.stopPropagation();
            if (!dragState.isDragging) {
              console.log("Handle clicked:", handle.key);
              setSelectedHandle(handle.key);
            }
          }}
        />
      );
    });
  };

  const handleContentsChange = (contents: string) => {
    // Preprocess the SVG to add data-region-id attributes
    const processedSVG = preprocessSVG(contents);

    if (svgContainerRef.current != null) {
      const handlePlaceCallback = selectedHandle 
        ? (position: { x: number; y: number }) => onHandlePlace(selectedHandle, position)
        : undefined;
      injectSVG(svgContainerRef.current, processedSVG, onElementClick, handlePlaceCallback);
    }

    // Save the preprocessed SVG to the form
    onContentsChange(processedSVG);
  };

  // Re-inject SVG when handle selection changes
  useEffect(() => {
    if (svgContainerRef.current != null && spec?.svg) {
      const handlePlaceCallback = selectedHandle 
        ? (position: { x: number; y: number }) => onHandlePlace(selectedHandle, position)
        : undefined;
      injectSVG(svgContainerRef.current, spec.svg, onElementClick, handlePlaceCallback);
    }
  }, [selectedHandle]);

  // Switch to handle mode when a handle is selected
  const effectivePlacementMode = selectedHandle ? 'handle' : 'region';

  return (
    <FileDrop
      onContentsChange={handleContentsChange}
      grow={1}
      enabled={spec.svg.length == 0}
    >
      <div 
        style={{ position: "relative", width: "100%", height: "100%" }}
        onMouseMove={(e) => {
          if (!dragState.isDragging || !dragState.handleKey) return;
          
          const svgElement = svgContainerRef.current?.querySelector('svg');
          if (!svgElement) return;
          
          const svgRect = svgElement.getBoundingClientRect();
          const svgDimensions = parseSVGDimensions(spec.svg);
          
          // Convert mouse position to SVG coordinates
          const x = svgDimensions.x + ((e.clientX - svgRect.left) / svgRect.width) * svgDimensions.width;
          const y = svgDimensions.y + ((e.clientY - svgRect.top) / svgRect.height) * svgDimensions.height;
          
          console.log("Dragging handle to:", { x, y });
          
          // Update handle position immediately
          onHandlePlace(dragState.handleKey, { x, y });
        }}
        onMouseUp={() => {
          if (dragState.isDragging) {
            console.log("Ending drag for handle:", dragState.handleKey);
            setDragState({ handleKey: null, isDragging: false, startPos: { x: 0, y: 0 } });
          }
        }}
        onMouseLeave={() => {
          // End drag if mouse leaves the container
          if (dragState.isDragging) {
            console.log("Mouse left container, ending drag");
            setDragState({ handleKey: null, isDragging: false, startPos: { x: 0, y: 0 } });
          }
        }}
      >
        <div
          ref={svgContainerRef}
          className={CSS.B("preview")}
          style={{ 
            width: "100%", 
            height: "100%", 
            minHeight: "400px",
            display: "flex",
            alignItems: "center",
            justifyContent: "center"
          }}
        />
        <div 
          ref={handleOverlayRef} 
          style={{ 
            position: "absolute", 
            inset: 0, 
            pointerEvents: "none",
            zIndex: 999
          }}
        >
          {renderHandleOverlays()}
        </div>
        {effectivePlacementMode === 'handle' && selectedHandle && (
          <div style={{
            position: "absolute",
            top: 8,
            right: 8,
            padding: "4px 8px",
            backgroundColor: "var(--pluto-primary-500)",
            color: "white",
            borderRadius: 4,
            fontSize: "12px",
            zIndex: 20
          }}>
            Click to place handle
          </div>
        )}
      </div>
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
  const [selectedHandle, setSelectedHandle] = useState<string>("");

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
    const currentHandles = form.get<{ key: string; position: { x: number; y: number } }[]>("data.handles").value;
    const newHandle: { key: string; position: { x: number; y: number } } = {
      key: `handle-${id.create()}`,
      position: { x: 50, y: 50 }, // Default center position
    };

    form.set("data.handles", [...currentHandles, newHandle]);
    setSelectedHandle(newHandle.key);
    // Clear region selection when switching to handle mode
    setSelectedRegion("");
  };

  const handleHandlePlace = (handleKey: string, position: { x: number; y: number }) => {
    const currentHandles = form.get<{ key: string; position: { x: number; y: number } }[]>("data.handles").value;
    const handleIndex = currentHandles.findIndex(h => h.key === handleKey);
    
    if (handleIndex !== -1) {
      const updatedHandles = [...currentHandles];
      updatedHandles[handleIndex] = { ...updatedHandles[handleIndex], position };
      form.set("data.handles", updatedHandles);
    }
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
              onChange={(value) => {
                setSelectedRegion(value);
                if (value) setSelectedHandle(""); // Clear handle selection when region is selected
              }}
              selectedState={selectedState}
              onAddRegion={addNewRegion}
            />
            {selectedHandle && (
              <Text.Text level="small" color={7}>
                Click on the SVG to place the selected handle
              </Text.Text>
            )}
            <HandleList
              value={selectedHandle}
              onChange={(value) => {
                setSelectedHandle(value);
                if (value) setSelectedRegion(""); // Clear region selection when handle is selected
              }}
              onAddHandle={addNewHandle}
            />
          </Flex.Box>
          <Form.Field<string> path="data.svg" showLabel={false} showHelpText={false}>
            {({ onChange }) => (
              <Preview
                selectedState={selectedState}
                selectedRegion={selectedRegion}
                selectedHandle={selectedHandle}
                onElementClick={handleElementClick}
                onContentsChange={onChange}
                onHandlePlace={handleHandlePlace}
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
