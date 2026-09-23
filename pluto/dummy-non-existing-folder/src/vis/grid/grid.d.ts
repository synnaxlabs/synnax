import { box, location, xy } from "@synnaxlabs/x";
import { z } from "zod";
export declare const regionZ: z.ZodObject<{
    key: z.ZodString;
    size: z.ZodNumber;
    order: z.ZodNumber;
    loc: z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>;
}, z.core.$strip>;
export declare const gridZ: z.ZodRecord<z.ZodString, z.ZodObject<{
    key: z.ZodString;
    size: z.ZodNumber;
    order: z.ZodNumber;
    loc: z.ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>;
}, z.core.$strip>>;
/**
 * An entry for a particular region in the grid, defined by a size, order, and location.
 * @property {string} key - The unique key for the region.
 * @property {number} size - The size of the region.
 * @property {number} order - The order of the region. A higher order means the region
 * will be positioned further away from the center of the container.
 * @property {location.Outer} loc - The location of the region.
 *
 * @example
 * const region: Region = {
 *  key: "x-axis",
 *  size: 50,
 *  order: 1,
 *  loc: "bottom",
 * };
 * // This region will be positioned on the bottom of the container.
 */
export type Region = z.input<typeof regionZ>;
/**
 * A uniform grid used to position elements in the outer regions of a container. This grid
 * is particularly useful for positioning axes and other elements that are not part of the
 * main visualization.
 *
 * @example
 * const grid: Grid = {
 *   "x-axis": {
 *     key: "x-axis",
 *     size: 50,
 *     order: 1,
 *     loc: "bottom",
 *  },
 *  "y-axis": {
 *     key: "y-axis",
 *     size: 50,
 *     order: 1,
 *     loc: "left",
 *  },
 *  // This axis will be positioned closer to the visualization because it has a lower
 *  // order than "y-axis".
 *  "y-axis-2": {
 *    key: "y-axis-2",
 *    size: 50,
 *    order: 0,
 *    loc: "left",
 *  },
 *  title: {
 *     key: "title",
 *     size: 50,
 *     order: 2,
 *     loc: "top",
 *  },
 */
export type Grid = z.input<typeof gridZ>;
/**
 * Extracts the regions for a particular location on the grid, sorted by order.
 * @returns The regions for the specified location.
 */
export declare const regions: (loc: location.Outer, grid: Grid) => Region[];
/**
 * Calculates the X and Y coordinates of the top-left corner of a region in the grid
 * based on a containing box.
 * @param container The container to calculate the position within.
 * @returns The X and Y coordinates for the region.
 */
export declare const position: (key: string, grid: Grid, container: box.Box) => xy.XY;
/**
 * Calculates the width and height for the visualization in the center of the grid
 * after accounting for all additional regions and the size of the container.
 * @param grid The grid to calculate the visualization box from.
 * @param container The container to calculate the visualization box within.
 * @returns The box for the visualization.
 */
export declare const visualizationBox: (grid: Grid, container: box.Box) => box.Box;
//# sourceMappingURL=grid.d.ts.map