import { schematic } from "@synnaxlabs/client";
import { type Diagram } from "../../vis/diagram";
export declare const buildParentOf: (configs: Record<string, schematic.ElementConfig>) => Map<string, string>;
/** canGroup returns whether the selection resolves to two or more outermost symbols. */
export declare const canGroup: (selected: readonly string[], nodes: readonly schematic.Node[], parentOf: Map<string, string>) => boolean;
export interface CreateParams {
    selected: readonly string[];
    nodes: readonly schematic.Node[];
    configs: Record<string, schematic.ElementConfig>;
}
export interface CreateResult {
    actions: schematic.Action[];
    /** selection lists the keys to select: the new group first, then its members. */
    selection: string[];
}
/**
 * createActions builds the one-batch group action: a setNode inserting a group
 * whose members are the outermost groups (or ungrouped symbols) the selection
 * resolves to. The box sizes itself from its members at render time. Returns
 * null when fewer than two resolve.
 */
export declare const createActions: ({ selected, nodes, configs, }: CreateParams) => CreateResult | null;
/**
 * remapMembers rewrites a pasted group config's members onto the pasted keys,
 * dropping members that were not pasted. Non-group configs pass through.
 */
export declare const remapMembers: (config: schematic.ElementConfig | undefined, remap: Record<string, string>) => schematic.ElementConfig | undefined;
/**
 * withMembers returns the keys plus every selected group's members, recursively.
 */
export declare const withMembers: (keys: readonly string[], configs: Record<string, schematic.ElementConfig>) => string[];
/**
 * fanOutMoves applies a moved group's delta to every symbol inside it. Keys
 * already moved in the batch are skipped. Zero deltas still fan out, so every
 * drag frame targets the same keys and coalesces into one undo step.
 */
export declare const fanOutMoves: (current: schematic.Schematic, actions: schematic.Action[]) => schematic.Action[];
/**
 * lockMembers marks grouped symbols and locked group boxes as non-draggable.
 * Returns the input unchanged when there are no groups.
 */
export declare const lockMembers: (nodes: schematic.Node[], parentOf: Map<string, string>, configs: Record<string, schematic.ElementConfig>) => Diagram.Node[];
/**
 * drillIn returns the member and, for a nested group, its members. Null when
 * the key is in no group or a containing group is locked.
 */
export declare const drillIn: (key: string, parentOf: Map<string, string>, configs: Record<string, schematic.ElementConfig>) => string[] | null;
/** shielded returns the keys that sit inside a locked group. */
export declare const shielded: (keys: readonly string[], parentOf: Map<string, string>, configs: Record<string, schematic.ElementConfig>) => Set<string>;
/** closure resolves each key to its outermost group and includes its members. */
export declare const closure: (keys: readonly string[], parentOf: Map<string, string>, configs: Record<string, schematic.ElementConfig>) => string[];
/** canUngroup returns whether the selection includes a group. */
export declare const canUngroup: (selected: readonly string[], configs: Record<string, schematic.ElementConfig>) => boolean;
export interface UngroupResult {
    actions: schematic.Action[];
    /** freed lists the removed groups' members, nested contents included. */
    freed: string[];
}
/**
 * ungroupActions builds the one-batch ungroup action: removes the selected
 * groups and the immediate parents of selected members, promoting a removed
 * group's members into the closest enclosing group that remains. Returns null
 * when the selection touches no group.
 */
export declare const ungroupActions: (selected: readonly string[], configs: Record<string, schematic.ElementConfig>) => UngroupResult | null;
//# sourceMappingURL=group.d.ts.map