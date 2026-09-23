import { type destructor } from "@synnaxlabs/x";
import { type CreateOptions, type Factory } from "../factory";
import { type Spec, type Telem } from "../telem";
export declare const registerInstance: (key: string, instance: Telem) => destructor.Destructor;
export declare const TEST_SINK_TYPE = "test-sink";
export declare const TEST_SOURCE_TYPE = "test-source";
export declare class TestFactory implements Factory {
    type: string;
    create(spec: Spec, _options?: CreateOptions): Telem | null;
}
//# sourceMappingURL=factory.d.ts.map