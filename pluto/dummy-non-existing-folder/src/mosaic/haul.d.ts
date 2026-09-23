import { Haul } from "@synnaxlabs/lyra/haul";
import { type record } from "@synnaxlabs/x";
/**
 * This type should be used when the user wants to drop a tab in the mosaic.
 * Dropping an item with this signature will call the {@link Frame} onDrop handler.
 */
export declare const HAUL_DROP_TYPE = "pluto_mosaic_tab_drop";
export type TabDropHaulItem = Haul.Item<typeof HAUL_DROP_TYPE, string, record.Unknown | undefined>;
/**
 * @param data - Opaque payload attached by the drag source and handed back to the
 * {@link Frame} onDrop handler. A mosaic in another window is a separate drop target
 * with its own state, so anything the destination needs about the tab's origin travels
 * here.
 */
export declare const createTabDropHaulItem: (tabKey: string, elementID?: string, data?: record.Unknown) => TabDropHaulItem;
export declare const isTabDropHaulItem: (item: Haul.Item) => item is TabDropHaulItem;
export declare const filterTabDropHaulItems: (items: Haul.Item[]) => TabDropHaulItem[];
export declare const canDropTabDropHaulItem: Haul.CanDrop;
/** This type should be used when the user wants to create a new tab in the mosaic.
Dropping an item with this signature will call the {@link Frame} onCreate handler. */
export declare const HAUL_CREATE_TYPE = "pluto_mosaic_tab_create";
export type TabCreateHaulItem = Haul.Item<typeof HAUL_CREATE_TYPE, string, undefined>;
export declare const createTabCreateHaulItem: (tabKey: string) => TabCreateHaulItem;
export declare const isTabCreateHaulItem: (item: Haul.Item) => item is TabCreateHaulItem;
export declare const filterTabCreateHaulItems: (items: Haul.Item[]) => TabCreateHaulItem[];
export declare const canDropTabCreateHaulItem: Haul.CanDrop;
//# sourceMappingURL=haul.d.ts.map