import "./Mosaic.css";
import { type ReactElement } from "react";
/**
 * Shield covers a leaf's content while a tab or OS file drag is in flight so drag
 * events reach the leaf instead of being swallowed by embedded content (canvases,
 * iframes). Render it inside the leaf's content region — a positioned container that
 * does NOT include the tab strip. Covering a dragged tab mid-dragstart cancels the
 * native drag, so the shield must never overlap the strip.
 */
export declare const Shield: () => ReactElement | null;
//# sourceMappingURL=Shield.d.ts.map