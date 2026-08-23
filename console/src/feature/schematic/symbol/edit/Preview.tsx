// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { type schematic } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  Icon,
  Schematic,
  Text,
  Theming,
  TimeSpan,
  Triggers,
} from "@synnaxlabs/pluto";
import { box, id, type xy } from "@synnaxlabs/x";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { FileDrop } from "@/feature/schematic/symbol/edit/FileDrop";
import { HandleOverlay } from "@/feature/schematic/symbol/edit/Handles";
import {
  FLATTENED_ZOOM_TRIGGERS,
  ZOOM_TRIGGERS,
} from "@/feature/schematic/symbol/edit/triggers";
import { CSS } from "@/platform/css";

const MIN_ZOOM = 0.1;
const MAX_ZOOM = 5;
const ZOOM_STEP = 1.2;
const ZOOM_REPEAT = TimeSpan.milliseconds(70);

interface PreviewProps {
  selectedState: string;
  selectedHandle?: string;
  onElementClick: (selector: string) => void;
  onContentsChange: (contents: string) => void;
  onHandleSelect: (handleKey: string) => void;
  onHandlePlace: (handleKey: string, position: { x: number; y: number }) => void;
}

export const Preview = ({
  selectedState,
  selectedHandle,
  onElementClick,
  onContentsChange,
  onHandlePlace,
  onHandleSelect,
}: PreviewProps): ReactElement | null => {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgWrapperRef = useRef<HTMLDivElement>(null);
  // State, not a ref: the theme provider takes the element as a prop, and a ref
  // filled after the first render would never re-render it into place.
  const [themeContainer, setThemeContainer] = useState<HTMLDivElement | null>(null);
  const spec = Form.useFieldValue<schematic.symbol.Spec>("data");
  const pan = Form.useField<xy.XY>("data.previewViewport.position");
  const zoom = Form.useField<number>("data.previewViewport.zoom");
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const isDark = Theming.use().key === "synnaxDark";
  const [isDarkMode, setIsDarkMode] = useState<boolean>(isDark);
  useEffect(() => setIsDarkMode(isDark), [isDark]);

  const svgElementRef = useRef<SVGSVGElement>(null);

  const resetViewport = () => {
    zoom.onChange(1);
    pan.onChange({ x: 0, y: 0 });
  };

  // A held chord steps on an interval, so the step reads the live zoom and setter
  // rather than the ones captured when the hold started.
  const zoomRef = useRef(zoom.value);
  zoomRef.current = zoom.value;
  const setZoomRef = useRef(zoom.onChange);
  setZoomRef.current = zoom.onChange;
  const stepZoom = useCallback(
    (factor: number) =>
      setZoomRef.current(
        Math.min(Math.max(zoomRef.current * factor, MIN_ZOOM), MAX_ZOOM),
      ),
    [],
  );

  const handleZoomIn = useCallback(() => stepZoom(ZOOM_STEP), [stepZoom]);
  const handleZoomOut = useCallback(() => stepZoom(1 / ZOOM_STEP), [stepZoom]);
  const handleResetZoom = () => resetViewport();

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    stepZoom(e.deltaY > 0 ? 0.9 : 1.1);
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button === 1 || (e.button === 0 && e.shiftKey)) {
      e.preventDefault();
      setIsDragging(true);
      setDragStart({
        x: e.clientX - pan.value.x,
        y: e.clientY - pan.value.y,
      });
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging)
      pan.onChange({ x: e.clientX - dragStart.x, y: e.clientY - dragStart.y });
  };

  const handleMouseUp = () => setIsDragging(false);

  const repeat = useRef<ReturnType<typeof setInterval>>(null);
  const stopRepeat = useCallback(() => {
    if (repeat.current == null) return;
    clearInterval(repeat.current);
    repeat.current = null;
  }, []);
  useEffect(() => stopRepeat, [stopRepeat]);

  Triggers.use({
    triggers: FLATTENED_ZOOM_TRIGGERS,
    enabled: Boolean(spec.svg),
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage === "end") return stopRepeat();
        if (stage !== "start") return;
        const mode = Triggers.determineMode(ZOOM_TRIGGERS, triggers);
        if (mode === "reset") return handleResetZoom();
        if (mode !== "in" && mode !== "out") return;
        const step = mode === "in" ? handleZoomIn : handleZoomOut;
        step();
        // A held chord keeps zooming, the way a held arrow key keeps scrolling.
        stopRepeat();
        repeat.current = setInterval(step, ZOOM_REPEAT.milliseconds);
      },
      [handleZoomIn, handleZoomOut, handleResetZoom, stopRepeat],
    ),
  });

  const onMount = (svgElement: SVGSVGElement) => {
    svgElementRef.current = svgElement;
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
    Array.from(svgElement.children[0].children).forEach(addInteractivity);
  };

  const setRenderContainer = Schematic.Node.Custom.useRender({
    orientation: "left",
    activeState: selectedState,
    externalScale: 1,
    spec,
    onMount,
  });
  const setContainer = useCallback(
    (el: HTMLDivElement | null) => {
      containerRef.current = el;
      setRenderContainer(el);
    },
    [setRenderContainer],
  );

  const form = Form.useContext();
  const handleContentsChange = (contents: string, filename?: string) => {
    const processedSVG = Schematic.Node.Region.normalizeSVG(contents);
    if (containerRef.current == null) return;
    onContentsChange(processedSVG);

    if (filename != null && form.get("name").value === "") form.set("name", filename);

    const parser = new DOMParser();
    const svgDoc = parser.parseFromString(processedSVG, "image/svg+xml");
    const svgElement = svgDoc.documentElement as unknown as SVGElement;
    const extractedRegions = Schematic.Node.Region.extract(svgElement);
    const states = form.get<schematic.symbol.State[]>("data.states").value;
    states.forEach((state) =>
      form.set(`data.states.${state.key}.regions`, extractedRegions),
    );
  };

  const fileDropEnabled = spec.svg.length === 0;
  let svgBox: box.Box = box.ZERO;
  if (svgElementRef.current != null) svgBox = box.construct(svgElementRef.current);

  const svgWrapperStyle = useMemo(
    () => ({
      transform: `translate(${pan.value.x}px, ${pan.value.y}px) scale(${zoom.value})`,
      transition: isDragging ? "none" : "transform 0.2s ease-out",
      cursor: isDragging ? "grabbing" : "default",
    }),
    [pan.value, zoom.value, isDragging],
  );

  return (
    <FileDrop
      onContentsChange={handleContentsChange}
      grow={1}
      enabled={fileDropEnabled}
    >
      <Theming.Provider
        el={themeContainer}
        theme={Theming.SYNNAX_THEMES[isDarkMode ? "synnaxDark" : "synnaxLight"]}
      >
        <Flex.Box
          ref={setThemeContainer}
          className={CSS.cls(
            CSS.B("schematic-preview-theme-container"),
            fileDropEnabled && CSS.M("hidden"),
          )}
          onWheel={handleWheel}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          background={0}
          rounded="small"
        >
          {spec.svg.length > 0 && (
            <Flex.Box x className={CSS.B("schematic-preview-controls")}>
              <Text.Text level="small" color={9}>
                {Math.round(zoom.value * 100)}%
              </Text.Text>
              <Button.Button
                variant="text"
                size="small"
                onClick={() => setIsDarkMode(!isDarkMode)}
              >
                {isDarkMode ? <Icon.DarkMode /> : <Icon.LightMode />}
              </Button.Button>
              <Flex.Box pack x>
                <Button.Button
                  onClick={handleZoomOut}
                  size="small"
                  tooltip={
                    <Triggers.Text trigger={ZOOM_TRIGGERS.modes.out[0]} level="small">
                      Zoom out
                    </Triggers.Text>
                  }
                >
                  <Icon.Subtract />
                </Button.Button>
                <Button.Button
                  onClick={handleZoomIn}
                  size="small"
                  tooltip={
                    <Triggers.Text trigger={ZOOM_TRIGGERS.modes.in[0]} level="small">
                      Zoom in
                    </Triggers.Text>
                  }
                >
                  <Icon.Add />
                </Button.Button>
                <Button.Button
                  onClick={handleResetZoom}
                  size="small"
                  tooltip={
                    <Triggers.Text trigger={ZOOM_TRIGGERS.modes.reset[0]} level="small">
                      Reset zoom
                    </Triggers.Text>
                  }
                >
                  <Icon.Expand />
                </Button.Button>
              </Flex.Box>
            </Flex.Box>
          )}
          <Flex.Box
            center
            ref={svgWrapperRef}
            className={CSS.B("schematic-preview-svg-wrapper")}
            style={svgWrapperStyle}
          >
            <div className={CSS.B("schematic-preview-svg-wrapper-inner")}>
              <HandleOverlay
                handles={spec.handles}
                selectedHandle={selectedHandle}
                svgBox={svgBox}
                onSelect={onHandleSelect}
                onDrag={onHandlePlace}
              />
              <div ref={setContainer} className={CSS.B("preview")}></div>
            </div>
          </Flex.Box>
        </Flex.Box>
      </Theming.Provider>
    </FileDrop>
  );
};
