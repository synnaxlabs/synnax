import { Aether } from "@/aether";
import { Align } from "@/align";
import { useGridEntry, useContext } from "@/vis/lineplot/LinePlot";
import { range } from "@/vis/lineplot/range/aether";
import { box, xy } from "@synnaxlabs/x";
import { ReactElement, useCallback, useRef } from "react";
import { Menu } from "@/menu";
import { RenderProp } from "@/util/renderProp";
import { SelectedState } from "@/vis/lineplot/range/aether/provider";
import { useSyncedRef } from "@/hooks";

export interface ProviderProps {
  menu?: RenderProp<range.SelectedState>;
}

export const Provider = Aether.wrap<ProviderProps>(
  "Annotation.Provider",
  ({ aetherKey, menu, ...props }): ReactElement => {
    const { setViewport, setHold } = useContext("Range.Provider");
    const [, { hovered, count }, setState] = Aether.use({
      aetherKey,
      type: range.Provider.TYPE,
      schema: range.providerStateZ,
      initialState: {
        ...props,
        cursor: null,
        hovered: null,
        count: 0,
      },
    });
    const gridStyle = useGridEntry(
      {
        key: aetherKey,
        loc: "top",
        size: count > 0 ? 32 : 0,
        order: "last",
      },
      "Annotation.Provider",
    );

    const menuProps = Menu.useContextMenu();
    const visibleRef = useSyncedRef(menuProps.visible);

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
            if (!visibleRef.current) setState((state) => ({ ...state, cursor: null }));
          },
          { once: true },
        );
      },
      [setState],
    );

    return (
      <Menu.ContextMenu
        style={{
          ...gridStyle,
          cursor: hovered != null ? "pointer" : "default",
        }}
        {...menuProps}
        menu={() => {
          if (menu == null || hovered == null) return null;
          return menu(hovered);
        }}
      >
        <Align.Space
          style={{ width: "100%", height: "100%" }}
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
              setHold(true);
            }
          }}
          onMouseEnter={handleMouseEnter}
        />
      </Menu.ContextMenu>
    );
  },
);
