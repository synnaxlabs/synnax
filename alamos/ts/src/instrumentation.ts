import { Tracer } from "@/trace";

export class Instrumentation {
    readonly T: Tracer

    constructor(tracer: Tracer) {
        this.T = tracer
    }
}