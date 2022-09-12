"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.RUNTIME = exports.Runtime = void 0;
var Runtime;
(function (Runtime) {
    Runtime["Browser"] = "browser";
    Runtime["Node"] = "node";
})(Runtime = exports.Runtime || (exports.Runtime = {}));
const detectRuntime = () => {
    if (typeof process !== 'undefined' &&
        process.versions != null &&
        process.versions.node != null) {
        return Runtime.Node;
    }
    if (typeof window !== 'undefined' && typeof window.document !== 'undefined') {
        return Runtime.Browser;
    }
    console.warn('Freighter unable to safely detect runtime, assuming browser');
    return Runtime.Browser;
};
exports.RUNTIME = detectRuntime();
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicnVudGltZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvcnVudGltZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFBQSxJQUFZLE9BR1g7QUFIRCxXQUFZLE9BQU87SUFDakIsOEJBQW1CLENBQUE7SUFDbkIsd0JBQWEsQ0FBQTtBQUNmLENBQUMsRUFIVyxPQUFPLEdBQVAsZUFBTyxLQUFQLGVBQU8sUUFHbEI7QUFFRCxNQUFNLGFBQWEsR0FBRyxHQUFZLEVBQUU7SUFDbEMsSUFDRSxPQUFPLE9BQU8sS0FBSyxXQUFXO1FBQzlCLE9BQU8sQ0FBQyxRQUFRLElBQUksSUFBSTtRQUN4QixPQUFPLENBQUMsUUFBUSxDQUFDLElBQUksSUFBSSxJQUFJLEVBQzdCO1FBQ0EsT0FBTyxPQUFPLENBQUMsSUFBSSxDQUFDO0tBQ3JCO0lBRUQsSUFBSSxPQUFPLE1BQU0sS0FBSyxXQUFXLElBQUksT0FBTyxNQUFNLENBQUMsUUFBUSxLQUFLLFdBQVcsRUFBRTtRQUMzRSxPQUFPLE9BQU8sQ0FBQyxPQUFPLENBQUM7S0FDeEI7SUFFRCxPQUFPLENBQUMsSUFBSSxDQUFDLDZEQUE2RCxDQUFDLENBQUM7SUFDNUUsT0FBTyxPQUFPLENBQUMsT0FBTyxDQUFDO0FBQ3pCLENBQUMsQ0FBQztBQUVXLFFBQUEsT0FBTyxHQUFHLGFBQWEsRUFBRSxDQUFDIn0=