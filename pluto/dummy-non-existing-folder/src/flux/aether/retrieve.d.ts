import { query, type Synnax as Client } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";
export interface RetrieveParams<Query extends query.Params> {
    client: Client;
    query: Query;
}
/**
 * A domain query definition: fetch = `retrieve`, live updates = `onChange`, snapshot =
 * `getCached`, all delegating to the domain client's read surface. React-free, so one
 * definition serves both the React and aether flux bindings.
 */
export interface Definition<Query extends query.Params, Data extends query.Data> {
    name: string;
    retrieve: (params: RetrieveParams<Query>) => Promise<Data>;
    onChange?: (params: RetrieveParams<Query>, handler: query.ChangeHandler<Data>) => destructor.Destructor;
    getCached?: (params: RetrieveParams<Query>) => query.Cached<Data> | undefined;
}
export interface RetrieveArgs<Query extends query.Params, Data extends query.Data> {
    definition: Definition<Query, Data>;
    /** Fires whenever the answer changes: a live value, a deletion, or undefined when
     * nothing is cached. Invalidation gaps are held, not forwarded. */
    onChange?: (result: query.Cached<Data> | undefined) => void;
    /** Receives fetch failures. Not-found resolves to an undefined answer instead. */
    onError?: (error: Error) => void;
}
/**
 * Holds one query's answer live for an aether component: seeds synchronously from the
 * domain cache, fetches on a miss, subscribes for changes, and pushes updates through
 * `onChange`. Call {@link update} from `afterUpdate` and {@link close} from
 * `afterDelete`. Repeated updates with an equal query and client are no-ops.
 */
export declare class Retrieve<Query extends query.Params, Data extends query.Data> {
    private readonly definition;
    private readonly notify?;
    private readonly onError?;
    private disconnect;
    private client;
    private query;
    private generation;
    private result;
    constructor({ definition, onChange, onError }: RetrieveArgs<Query, Data>);
    /** The live answer, or undefined while absent, deleted, or not yet fetched. */
    get value(): Data | undefined;
    /** The raw answer: live, deleted, or undefined when nothing is cached. */
    get cached(): query.Cached<Data> | undefined;
    /**
     * Points the observer at a query. Tears down the previous subscription, seeds from
     * the cache, and fetches on a miss. A null client clears the answer.
     */
    update(client: Client | null, q: Query): void;
    /** Refetches the current query, superseding any fetch in flight. */
    refetch(): void;
    /** Never rejects: a failure is routed to onError and a stale generation dropped. */
    private fetch;
    private set;
    private teardown;
    /** Stops the subscription and drops in-flight fetches. Safe to call twice. */
    close(): void;
}
//# sourceMappingURL=retrieve.d.ts.map