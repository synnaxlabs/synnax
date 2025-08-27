import { type schematic } from "@synnaxlabs/client";
import { Button, Flex, Form, Icon, Text, Theming } from "@synnaxlabs/pluto";
import { box, id } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { CSS } from "@/css";
import { FileDrop } from "@/schematic/symbols/FileDrop";
import { HandleOverlay } from "@/schematic/symbols/Handles";

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

export const Preview = ({
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
  const themeContainerRef = useRef<HTMLDivElement>(null);
  const spec = Form.useFieldValue<schematic.symbol.Spec>("data");
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [isDarkMode, setIsDarkMode] = useState(false);
  const scale = Form.useFieldValue<number>("data.scale");

  const resetZoom = () => {
    setZoom(1);
    setPan({ x: 0, y: 0 });
  };

  const handleZoomIn = () => setZoom((z) => Math.min(z * 1.2, 5));
  const handleZoomOut = () => setZoom((z) => Math.max(z / 1.2, 0.1));
  const handleResetZoom = () => resetZoom();

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

  const handleContentsChange = (contents: string) => {
    const processedSVG = preprocessSVG(contents);
    if (containerRef.current == null) return;
    onContentsChange(processedSVG);
  };

  if (containerRef.current != null && spec.svg.length > 0) {
    injectSVG(containerRef.current, spec.svg);
    applyLivePreview();
  }

  let svgBox: box.Box = box.ZERO;
  if (svgElementRef.current) svgBox = box.construct(svgElementRef.current);

  const fileDropEnabled = spec.svg.length === 0;
  return (
    <FileDrop
      onContentsChange={handleContentsChange}
      grow={1}
      enabled={fileDropEnabled}
    >
      <Theming.Provider
        el={themeContainerRef.current}
        theme={Theming.SYNNAX_THEMES[isDarkMode ? "synnaxDark" : "synnaxLight"]}
      >
        <Flex.Box
          ref={themeContainerRef}
          style={{
            position: "relative",
            width: "100%",
            height: "100%",
            overflow: "hidden",
            display: fileDropEnabled ? "none" : "flex",
          }}
          onWheel={handleWheel}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          background={0}
          rounded={1}
        >
          {spec.svg.length > 0 && (
            <Flex.Box
              x
              style={{
                position: "absolute",
                top: 16,
                right: 16,
                zIndex: 1000,
              }}
            >
              <Text.Text level="small" color={7}>
                {Math.round(zoom * 100)}%
              </Text.Text>
              <Button.Button
                variant="text"
                size="small"
                onClick={() => setIsDarkMode(!isDarkMode)}
              >
                {isDarkMode ? <Icon.DarkMode /> : <Icon.LightMode />}
              </Button.Button>
              <Flex.Box pack x>
                <Button.Button onClick={handleZoomOut} size="small" tooltip="Zoom Out">
                  <Icon.Subtract />
                </Button.Button>
                <Button.Button onClick={handleZoomIn} size="small" tooltip="Zoom In">
                  <Icon.Add />
                </Button.Button>
                <Button.Button
                  onClick={handleResetZoom}
                  size="small"
                  tooltip="Reset Zoom"
                >
                  <Icon.Expand />
                </Button.Button>
              </Flex.Box>
            </Flex.Box>
          )}
          <Flex.Box
            center
            ref={svgWrapperRef}
            style={{
              transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
              transformOrigin: "center",
              transition: isDragging ? "none" : "transform 0.2s ease-out",
              cursor: isDragging ? "grabbing" : "default",
            }}
            rounded={1}
          >
            <div
              style={{
                position: "relative",
                transform: `scale(${scale})`,
                transformOrigin: "center",
              }}
            >
              <HandleOverlay
                handles={spec.handles}
                selectedHandle={selectedHandle}
                svgBox={svgBox}
                onSelect={onHandleSelect}
                onDrag={onHandlePlace}
              />
              <div
                ref={containerRef}
                className={CSS.B("preview")}
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              ></div>
            </div>
          </Flex.Box>
        </Flex.Box>
      </Theming.Provider>
    </FileDrop>
  );
};
