import { arc } from "@synnaxlabs/client";
import { crdt } from "@synnaxlabs/x";
import { type Diff, diff } from "../../code/text";
export { type Diff, diff };
/** TextChange is the subset of an editor content change the binding consumes: a UTF-16
 * offset and length into the previous value, plus the inserted text. It matches Monaco's
 * IModelContentChange. */
export interface TextChange {
    rangeOffset: number;
    rangeLength: number;
    text: string;
}
/** changesToDiffs converts editor content changes (UTF-16 offsets into prev) into
 * code-point Diffs, ordered highest-offset-first so that applying each change in turn
 * does not shift the offsets of the changes not yet applied. prev must be the document
 * value the offsets are relative to (the value before the change batch). */
export declare const changesToDiffs: (prev: string, changes: readonly TextChange[]) => Diff[];
/** CollabText binds a replicated text document to a plain-string editing surface. It
 * bootstraps from a server snapshot so it shares the server's id space, translates
 * whole-value edits into CRDT operations to broadcast, applies remote operations, and
 * always exposes the materialized string. It does not know about the transport or the
 * editor: callers feed it new values and remote operations and read back the value. */
export declare class CollabText {
    private readonly doc;
    private constructor();
    /** bootstrap creates a document from a server snapshot so the client's edits compose
     * with every other editor's in the same id space. */
    static bootstrap(snapshot: {
        inserts: crdt.Insert[];
        deletes: crdt.Delete[];
    }): CollabText;
    /** value returns the current materialized text. */
    value(): string;
    /** applyChanges applies a batch of code-point changes to the document and returns the
     * operations to broadcast. The changes must be ordered highest-index-first (as
     * changesToDiffs produces) so each applies against indices the others have not shifted.
     * Returns an empty array when the batch is a no-op. */
    applyChanges(changes: Diff[]): arc.Action[];
    /** edit reconciles the document to next via a single whole-value diff and returns the
     * operations that describe the change. Prefer applyChanges with precise editor changes;
     * this is a convenience for callers that only have the new value. */
    edit(next: string): arc.Action[];
    /** applyRemote integrates operations produced by other editors. */
    applyRemote(actions: arc.Action[]): void;
    /** sync integrates every operation in snapshot that the document has not already seen,
     * pulling in remote edits while preserving this replica's authoring state. Integration
     * is idempotent, so re-syncing the full snapshot after every change is safe. */
    sync(snapshot: {
        inserts: crdt.Insert[];
        deletes: crdt.Delete[];
    }): void;
}
//# sourceMappingURL=collab.d.ts.map