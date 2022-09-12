export var Runtime;
(function (Runtime) {
    Runtime["Browser"] = "browser";
    Runtime["Node"] = "node";
})(Runtime || (Runtime = {}));
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
export const RUNTIME = detectRuntime();
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicnVudGltZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvcnVudGltZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxNQUFNLENBQU4sSUFBWSxPQUdYO0FBSEQsV0FBWSxPQUFPO0lBQ2pCLDhCQUFtQixDQUFBO0lBQ25CLHdCQUFhLENBQUE7QUFDZixDQUFDLEVBSFcsT0FBTyxLQUFQLE9BQU8sUUFHbEI7QUFFRCxNQUFNLGFBQWEsR0FBRyxHQUFZLEVBQUU7SUFDbEMsSUFDRSxPQUFPLE9BQU8sS0FBSyxXQUFXO1FBQzlCLE9BQU8sQ0FBQyxRQUFRLElBQUksSUFBSTtRQUN4QixPQUFPLENBQUMsUUFBUSxDQUFDLElBQUksSUFBSSxJQUFJLEVBQzdCO1FBQ0EsT0FBTyxPQUFPLENBQUMsSUFBSSxDQUFDO0tBQ3JCO0lBRUQsSUFBSSxPQUFPLE1BQU0sS0FBSyxXQUFXLElBQUksT0FBTyxNQUFNLENBQUMsUUFBUSxLQUFLLFdBQVcsRUFBRTtRQUMzRSxPQUFPLE9BQU8sQ0FBQyxPQUFPLENBQUM7S0FDeEI7SUFFRCxPQUFPLENBQUMsSUFBSSxDQUFDLDZEQUE2RCxDQUFDLENBQUM7SUFDNUUsT0FBTyxPQUFPLENBQUMsT0FBTyxDQUFDO0FBQ3pCLENBQUMsQ0FBQztBQUVGLE1BQU0sQ0FBQyxNQUFNLE9BQU8sR0FBRyxhQUFhLEVBQUUsQ0FBQyJ9