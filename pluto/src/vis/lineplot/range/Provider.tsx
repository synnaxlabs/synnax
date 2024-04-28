import { Aether } from "@/aether";
import { Align } from "@/align";
import { useGridEntry, useContext } from "@/vis/lineplot/LinePlot";
import { range } from "@/vis/lineplot/range/aether";
import { box, xy } from "@synnaxlabs/x";
import { ReactElement, useCallback } from "react";
import { z } from "zod";

interface ProviderProps {}

export const Provider = Aether.wrap<ProviderProps>(
  "Annotation.Provider",
  ({ aetherKey, ...props }): ReactElement => {
    const { setViewport } = useContext("Range.Provider");
    const [, { hovered }, setState] = Aether.use({
      aetherKey,
      type: range.Provider.TYPE,
      schema: range.providerStateZ,
      initialState: {
        ...props,
        cursor: null,
        hovered: null,
      },
    });
    const gridStyle = useGridEntry(
      {
        key: aetherKey,
        loc: "top",
        size: 32,
        order: "last",
      },
      "Annotation.Provider",
    );

    const handleMouseEnter: React.MouseEventHandler<HTMLDivElement> = useCallback(
      (e) => {
        // add an event listener for the movement until it leaves
        const handleMouseMove = (e: MouseEvent) => {
          setState((state) => ({ ...state, cursor: { x: e.clientX, y: e.clientY } }));
        };
        const target = e.currentTarget;
        target.addEventListener("mousemove", handleMouseMove);
        target.addEventListener(
          "mouseleave",
          () => {
            target.removeEventListener("mousemove", handleMouseMove);
            setState((state) => ({ ...state, cursor: null }));
          },
          { once: true },
        );
      },
      [setState],
    );

    return (
      <Align.Space
        style={{
          ...gridStyle,
          cursor: hovered != null ? "pointer" : "default",
        }}
        onClick={() => {
          if (hovered != null) {
            setViewport({
              box: box.construct(
                { x: hovered.viewport.lower, y: 0 },
                { x: hovered.viewport.upper, y: 1 },
              ),
              mode: "zoom",
              cursor: xy.ZERO,
              stage: "start",
            });
          }
        }}
        onMouseEnter={handleMouseEnter}
      />
    );
  },
);
