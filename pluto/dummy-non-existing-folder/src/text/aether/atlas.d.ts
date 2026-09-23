import { color, type dimensions } from "@synnaxlabs/x";
export interface AtlasProps {
    font: string;
    textColor: color.Crude;
    characters?: string;
}
/**
 * @desc a text atlas that allows for efficient caching and rendering of monospaced
 * characters.
 */
export declare class MonospacedAtlas {
    private readonly atlas;
    private readonly charDims;
    private readonly charMap;
    private readonly cols;
    /** Height of one grid cell, tall enough to hold any glyph in the set. */
    private readonly cellHeight;
    /** Distance from the top of a cell to the baseline the glyph is drawn on. */
    private readonly baselineOffset;
    /** Drop from the origin each text baseline sets to the alphabetic baseline. */
    private readonly baselineShifts;
    private static readonly DEFAULT_CHARS;
    constructor(props: AtlasProps);
    fillText(ctx: OffscreenCanvasRenderingContext2D, text: string, x: number, y: number): void;
    measureText(text: string): dimensions.Dimensions;
}
/** A registry for caching atlases for use across multiple components. */
export declare class AtlasRegistry {
    private readonly atlases;
    constructor();
    /**
     * @returns at atlas from the registry compatible with the given props. If the
     * atlas does not exist in the registry, it is created and added to the registry.
     */
    get(props: AtlasProps): MonospacedAtlas;
}
//# sourceMappingURL=atlas.d.ts.map