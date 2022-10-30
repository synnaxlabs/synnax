"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ontologyResourceSchema = exports.ontologySchemaSchema = exports.ontologySchemaFieldSchema = exports.OntologyRoot = exports.OntologyID = exports.ontologyIdSchema = exports.OntologyResourceType = void 0;
const zod_1 = require("zod");
var OntologyResourceType;
(function (OntologyResourceType) {
    OntologyResourceType["Builtin"] = "builtin";
    OntologyResourceType["Cluster"] = "cluster";
    OntologyResourceType["Channel"] = "channel";
    OntologyResourceType["Node"] = "node";
})(OntologyResourceType = exports.OntologyResourceType || (exports.OntologyResourceType = {}));
exports.ontologyIdSchema = zod_1.z.object({
    type: zod_1.z.nativeEnum(OntologyResourceType),
    key: zod_1.z.string(),
});
class OntologyID {
    constructor(type, key) {
        this.type = type;
        this.key = key;
    }
    toString() {
        return `${this.type}:${this.key}`;
    }
    static parseString(str) {
        const [type, key] = str.split(':');
        return new OntologyID(type, key);
    }
}
exports.OntologyID = OntologyID;
exports.OntologyRoot = new OntologyID(OntologyResourceType.Builtin, 'root');
exports.ontologySchemaFieldSchema = zod_1.z.object({
    type: zod_1.z.number(),
});
exports.ontologySchemaSchema = zod_1.z.object({
    type: zod_1.z.nativeEnum(OntologyResourceType),
    fields: zod_1.z.record(exports.ontologySchemaFieldSchema),
});
exports.ontologyResourceSchema = zod_1.z.object({
    id: exports.ontologyIdSchema.transform((id) => new OntologyID(id.type, id.key)),
    entity: zod_1.z.object({
        schema: exports.ontologySchemaSchema,
        name: zod_1.z.string(),
        data: zod_1.z.record(zod_1.z.unknown()),
    }),
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicGF5bG9hZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uLy4uL3NyYy9saWIvb250b2xvZ3kvcGF5bG9hZC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFBQSw2QkFBd0I7QUFFeEIsSUFBWSxvQkFLWDtBQUxELFdBQVksb0JBQW9CO0lBQzlCLDJDQUFtQixDQUFBO0lBQ25CLDJDQUFtQixDQUFBO0lBQ25CLDJDQUFtQixDQUFBO0lBQ25CLHFDQUFhLENBQUE7QUFDZixDQUFDLEVBTFcsb0JBQW9CLEdBQXBCLDRCQUFvQixLQUFwQiw0QkFBb0IsUUFLL0I7QUFFWSxRQUFBLGdCQUFnQixHQUFHLE9BQUMsQ0FBQyxNQUFNLENBQUM7SUFDdkMsSUFBSSxFQUFFLE9BQUMsQ0FBQyxVQUFVLENBQUMsb0JBQW9CLENBQUM7SUFDeEMsR0FBRyxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUU7Q0FDaEIsQ0FBQyxDQUFDO0FBRUgsTUFBYSxVQUFVO0lBSXJCLFlBQVksSUFBMEIsRUFBRSxHQUFXO1FBQ2pELElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBQ2pCLElBQUksQ0FBQyxHQUFHLEdBQUcsR0FBRyxDQUFDO0lBQ2pCLENBQUM7SUFFRCxRQUFRO1FBQ04sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLElBQUksSUFBSSxDQUFDLEdBQUcsRUFBRSxDQUFDO0lBQ3BDLENBQUM7SUFFRCxNQUFNLENBQUMsV0FBVyxDQUFDLEdBQVc7UUFDNUIsTUFBTSxDQUFDLElBQUksRUFBRSxHQUFHLENBQUMsR0FBRyxHQUFHLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ25DLE9BQU8sSUFBSSxVQUFVLENBQUMsSUFBNEIsRUFBRSxHQUFHLENBQUMsQ0FBQztJQUMzRCxDQUFDO0NBQ0Y7QUFqQkQsZ0NBaUJDO0FBRVksUUFBQSxZQUFZLEdBQUcsSUFBSSxVQUFVLENBQ3hDLG9CQUFvQixDQUFDLE9BQU8sRUFDNUIsTUFBTSxDQUNQLENBQUM7QUFFVyxRQUFBLHlCQUF5QixHQUFHLE9BQUMsQ0FBQyxNQUFNLENBQUM7SUFDaEQsSUFBSSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUU7Q0FDakIsQ0FBQyxDQUFDO0FBSVUsUUFBQSxvQkFBb0IsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQzNDLElBQUksRUFBRSxPQUFDLENBQUMsVUFBVSxDQUFDLG9CQUFvQixDQUFDO0lBQ3hDLE1BQU0sRUFBRSxPQUFDLENBQUMsTUFBTSxDQUFDLGlDQUF5QixDQUFDO0NBQzVDLENBQUMsQ0FBQztBQUlVLFFBQUEsc0JBQXNCLEdBQUcsT0FBQyxDQUFDLE1BQU0sQ0FBQztJQUM3QyxFQUFFLEVBQUUsd0JBQWdCLENBQUMsU0FBUyxDQUFDLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxJQUFJLFVBQVUsQ0FBQyxFQUFFLENBQUMsSUFBSSxFQUFFLEVBQUUsQ0FBQyxHQUFHLENBQUMsQ0FBQztJQUN2RSxNQUFNLEVBQUUsT0FBQyxDQUFDLE1BQU0sQ0FBQztRQUNmLE1BQU0sRUFBRSw0QkFBb0I7UUFDNUIsSUFBSSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUU7UUFDaEIsSUFBSSxFQUFFLE9BQUMsQ0FBQyxNQUFNLENBQUMsT0FBQyxDQUFDLE9BQU8sRUFBRSxDQUFDO0tBQzVCLENBQUM7Q0FDSCxDQUFDLENBQUMifQ==