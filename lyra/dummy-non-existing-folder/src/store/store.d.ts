import { type destructor, type record } from "@synnaxlabs/x";
export interface UseKeyedListenersReturn<K extends record.Key> {
    notifyListeners: (keys: K | K[]) => void;
    subscribe: (listener: () => void, key?: K) => destructor.Destructor;
}
export declare const useKeyedListeners: <K extends record.Key>() => UseKeyedListenersReturn<K>;
//# sourceMappingURL=store.d.ts.map