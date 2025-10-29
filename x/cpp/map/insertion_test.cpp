// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <memory>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/map/insertion.h"
#include "x/cpp/xjson/xjson.h"

TEST(InsertionTest, EmptyMap) {
    map::Insertion<int> m;
    EXPECT_TRUE(m.empty());
    EXPECT_EQ(m.count(), 0);
}

TEST(InsertionTest, SetAndGet) {
    map::Insertion<int> m;
    m.set("key1", 42);
    EXPECT_FALSE(m.empty());
    EXPECT_EQ(m.count(), 1);
    const int *value = m.get("key1");
    ASSERT_NE(value, nullptr);
    EXPECT_EQ(*value, 42);
}

TEST(InsertionTest, GetNonExistentKey) {
    map::Insertion<int> m;
    m.set("key1", 42);
    const int *value = m.get("key2");
    EXPECT_EQ(value, nullptr);
}

TEST(InsertionTest, SetOverwritesExistingKey) {
    map::Insertion<int> m;
    m.set("key1", 42);
    m.set("key1", 100);
    EXPECT_EQ(m.count(), 1);
    const int *value = m.get("key1");
    ASSERT_NE(value, nullptr);
    EXPECT_EQ(*value, 100);
}

TEST(InsertionTest, SetMoveSemantics) {
    map::Insertion<std::unique_ptr<int>> m;
    auto ptr = std::make_unique<int>(42);
    m.set("key1", std::move(ptr));
    EXPECT_EQ(ptr, nullptr);
    const std::unique_ptr<int> *value = m.get("key1");
    ASSERT_NE(value, nullptr);
    EXPECT_NE(*value, nullptr);
    EXPECT_EQ(**value, 42);
}

TEST(InsertionTest, SetMoveOverwrite) {
    map::Insertion<std::unique_ptr<int>> m;
    m.set("key1", std::make_unique<int>(42));
    m.set("key1", std::make_unique<int>(100));
    EXPECT_EQ(m.count(), 1);
    const std::unique_ptr<int> *value = m.get("key1");
    ASSERT_NE(value, nullptr);
    EXPECT_EQ(**value, 100);
}

TEST(InsertionTest, Contains) {
    map::Insertion<int> m;
    m.set("key1", 42);
    EXPECT_TRUE(m.contains("key1"));
    EXPECT_FALSE(m.contains("key2"));
}

TEST(InsertionTest, Erase) {
    map::Insertion<int> m;
    m.set("key1", 42);
    m.set("key2", 100);
    EXPECT_EQ(m.count(), 2);
    EXPECT_TRUE(m.erase("key1"));
    EXPECT_EQ(m.count(), 1);
    EXPECT_FALSE(m.contains("key1"));
    EXPECT_TRUE(m.contains("key2"));
}

TEST(InsertionTest, EraseNonExistentKey) {
    map::Insertion<int> m;
    m.set("key1", 42);
    EXPECT_FALSE(m.erase("key2"));
    EXPECT_EQ(m.count(), 1);
}

TEST(InsertionTest, Clear) {
    map::Insertion<int> m;
    m.set("key1", 42);
    m.set("key2", 100);
    m.set("key3", 200);
    EXPECT_EQ(m.count(), 3);
    m.clear();
    EXPECT_TRUE(m.empty());
    EXPECT_EQ(m.count(), 0);
}

TEST(InsertionTest, InsertionOrderPreserved) {
    map::Insertion<int> m;
    m.set("third", 3);
    m.set("first", 1);
    m.set("second", 2);
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(m.key_at(0), "third");
    EXPECT_EQ(m.at(0), 3);
    EXPECT_EQ(m.key_at(1), "first");
    EXPECT_EQ(m.at(1), 1);
    EXPECT_EQ(m.key_at(2), "second");
    EXPECT_EQ(m.at(2), 2);
}

TEST(InsertionTest, InsertionOrderAfterOverwrite) {
    map::Insertion<int> m;
    m.set("first", 1);
    m.set("second", 2);
    m.set("third", 3);
    m.set("second", 200);
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(m.key_at(0), "first");
    EXPECT_EQ(m.key_at(1), "second");
    EXPECT_EQ(m.at(1), 200);
    EXPECT_EQ(m.key_at(2), "third");
}

TEST(InsertionTest, InsertionOrderAfterErase) {
    map::Insertion<int> m;
    m.set("first", 1);
    m.set("second", 2);
    m.set("third", 3);
    m.erase("second");
    EXPECT_EQ(m.count(), 2);
    EXPECT_EQ(m.key_at(0), "first");
    EXPECT_EQ(m.key_at(1), "third");
}

TEST(InsertionTest, AtIndexAccess) {
    map::Insertion<int> m;
    m.set("key1", 42);
    m.set("key2", 100);
    EXPECT_EQ(m.at(0), 42);
    EXPECT_EQ(m.at(1), 100);
}

TEST(InsertionTest, AtIndexAccessConst) {
    map::Insertion<int> m;
    m.set("key1", 42);
    const map::Insertion<int> &const_m = m;
    EXPECT_EQ(const_m.at(0), 42);
}

TEST(InsertionTest, AtIndexAccessMutable) {
    map::Insertion<int> m;
    m.set("key1", 42);
    m.at(0) = 100;
    EXPECT_EQ(m.at(0), 100);
    const int *value = m.get("key1");
    EXPECT_EQ(*value, 100);
}

TEST(InsertionTest, GetMutablePointer) {
    map::Insertion<int> m;
    m.set("key1", 42);
    int *value = m.get("key1");
    ASSERT_NE(value, nullptr);
    *value = 100;
    EXPECT_EQ(m.at(0), 100);
}

TEST(InsertionTest, KeyAtIndexAccess) {
    map::Insertion<int> m;
    m.set("alpha", 1);
    m.set("beta", 2);
    m.set("gamma", 3);
    EXPECT_EQ(m.key_at(0), "alpha");
    EXPECT_EQ(m.key_at(1), "beta");
    EXPECT_EQ(m.key_at(2), "gamma");
}

TEST(InsertionTest, Reserve) {
    map::Insertion<int> m;
    m.reserve(100);
    for (int i = 0; i < 100; i++)
        m.set("key" + std::to_string(i), i);
    EXPECT_EQ(m.count(), 100);
}

TEST(InsertionTest, MultipleOperations) {
    map::Insertion<std::string> m;
    m.set("name", "Alice");
    m.set("city", "New York");
    m.set("country", "USA");
    EXPECT_EQ(m.count(), 3);
    m.set("city", "San Francisco");
    EXPECT_EQ(m.count(), 3);
    const std::string *city = m.get("city");
    EXPECT_EQ(*city, "San Francisco");
    m.erase("country");
    EXPECT_EQ(m.count(), 2);
    EXPECT_FALSE(m.contains("country"));
    m.clear();
    EXPECT_TRUE(m.empty());
}

TEST(InsertionTest, ComplexValueType) {
    struct Data {
        int id;
        std::string name;
        Data(int i, std::string n): id(i), name(std::move(n)) {}
    };
    map::Insertion<Data> m;
    m.set("first", Data(1, "Alice"));
    m.set("second", Data(2, "Bob"));
    const Data *data = m.get("first");
    ASSERT_NE(data, nullptr);
    EXPECT_EQ(data->id, 1);
    EXPECT_EQ(data->name, "Alice");
}

TEST(InsertionTest, StringKeys) {
    map::Insertion<int> m;
    m.set("", 0);
    m.set("a", 1);
    m.set("ab", 2);
    m.set("abc", 3);
    EXPECT_EQ(m.count(), 4);
    EXPECT_TRUE(m.contains(""));
    EXPECT_EQ(*m.get(""), 0);
    EXPECT_EQ(*m.get("abc"), 3);
}

TEST(InsertionTest, LargeMap) {
    map::Insertion<int> m;
    constexpr int num_items = 10000;
    for (int i = 0; i < num_items; i++)
        m.set("key" + std::to_string(i), i);
    EXPECT_EQ(m.count(), num_items);
    for (int i = 0; i < num_items; i++) {
        const int *value = m.get("key" + std::to_string(i));
        ASSERT_NE(value, nullptr);
        EXPECT_EQ(*value, i);
    }
}

TEST(InsertionTest, EraseFromMiddle) {
    map::Insertion<int> m;
    m.set("a", 1);
    m.set("b", 2);
    m.set("c", 3);
    m.set("d", 4);
    m.erase("b");
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(m.key_at(0), "a");
    EXPECT_EQ(m.key_at(1), "c");
    EXPECT_EQ(m.key_at(2), "d");
    EXPECT_EQ(m.at(0), 1);
    EXPECT_EQ(m.at(1), 3);
    EXPECT_EQ(m.at(2), 4);
}

TEST(InsertionTest, EraseFromBeginning) {
    map::Insertion<int> m;
    m.set("a", 1);
    m.set("b", 2);
    m.set("c", 3);
    m.erase("a");
    EXPECT_EQ(m.count(), 2);
    EXPECT_EQ(m.key_at(0), "b");
    EXPECT_EQ(m.key_at(1), "c");
}

TEST(InsertionTest, EraseFromEnd) {
    map::Insertion<int> m;
    m.set("a", 1);
    m.set("b", 2);
    m.set("c", 3);
    m.erase("c");
    EXPECT_EQ(m.count(), 2);
    EXPECT_EQ(m.key_at(0), "a");
    EXPECT_EQ(m.key_at(1), "b");
}

// ============================================================================
// JSON Parsing Tests
// ============================================================================

TEST(InsertionTest, JsonParseEmptyObject) {
    const json j = json::object();
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_TRUE(m.empty());
    EXPECT_EQ(m.count(), 0);
}

TEST(InsertionTest, JsonParseSimpleIntegers) {
    const json j = {{"x", 10}, {"y", 20}, {"z", 30}};
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(*m.get("x"), 10);
    EXPECT_EQ(*m.get("y"), 20);
    EXPECT_EQ(*m.get("z"), 30);
}

TEST(InsertionTest, JsonParseSimpleStrings) {
    const json j = {{"name", "Alice"}, {"city", "New York"}, {"country", "USA"}};
    xjson::Parser parser(j);
    map::Insertion<std::string> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(*m.get("name"), "Alice");
    EXPECT_EQ(*m.get("city"), "New York");
    EXPECT_EQ(*m.get("country"), "USA");
}

TEST(InsertionTest, JsonParseFloatValues) {
    const json j = {{"pi", 3.14159}, {"e", 2.71828}, {"golden_ratio", 1.61803}};
    xjson::Parser parser(j);
    map::Insertion<float> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_NEAR(*m.get("pi"), 3.14159f, 0.0001f);
    EXPECT_NEAR(*m.get("e"), 2.71828f, 0.0001f);
    EXPECT_NEAR(*m.get("golden_ratio"), 1.61803f, 0.0001f);
}

TEST(InsertionTest, JsonParseBoolValues) {
    const json j = {{"enabled", true}, {"disabled", false}, {"active", true}};
    xjson::Parser parser(j);
    map::Insertion<bool> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_TRUE(*m.get("enabled"));
    EXPECT_FALSE(*m.get("disabled"));
    EXPECT_TRUE(*m.get("active"));
}

TEST(InsertionTest, JsonParseInsertionOrderPreserved) {
    const json j = {{"third", 3}, {"first", 1}, {"second", 2}};
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    // nlohmann::json preserves insertion order
    EXPECT_EQ(m.key_at(0), "third");
    EXPECT_EQ(m.at(0), 3);
    EXPECT_EQ(m.key_at(1), "first");
    EXPECT_EQ(m.at(1), 1);
    EXPECT_EQ(m.key_at(2), "second");
    EXPECT_EQ(m.at(2), 2);
}

TEST(InsertionTest, JsonParseNotAnObject) {
    const json j = json::array({1, 2, 3});
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_GE(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "");
    EXPECT_EQ(err["message"], "Expected an object");
    EXPECT_TRUE(m.empty());
}

TEST(InsertionTest, JsonParseInvalidTypeSkipped) {
    const json j = {{"valid1", 42}, {"invalid", "not a number"}, {"valid2", 100}};
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_GE(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "invalid");
    // Valid entries should still be parsed
    EXPECT_EQ(m.count(), 2);
    EXPECT_EQ(*m.get("valid1"), 42);
    EXPECT_EQ(*m.get("valid2"), 100);
    EXPECT_EQ(m.get("invalid"), nullptr);
}

TEST(InsertionTest, JsonParseEmptyStrings) {
    const json j = {{"empty", ""}, {"space", " "}, {"text", "hello"}};
    xjson::Parser parser(j);
    map::Insertion<std::string> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(*m.get("empty"), "");
    EXPECT_EQ(*m.get("space"), " ");
    EXPECT_EQ(*m.get("text"), "hello");
}

TEST(InsertionTest, JsonParseFromString) {
    const std::string json_str = R"({
        "name": "test",
        "count": 42,
        "score": 95.5
    })";
    xjson::Parser parser(json_str);
    map::Insertion<json> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(m.get("name")->get<std::string>(), "test");
    EXPECT_EQ(m.get("count")->get<int>(), 42);
    EXPECT_NEAR(m.get("score")->get<double>(), 95.5, 0.001);
}

TEST(InsertionTest, JsonParseLargeObject) {
    json j = json::object();
    for (int i = 0; i < 1000; i++) {
        j["key" + std::to_string(i)] = i;
    }

    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 1000);

    for (int i = 0; i < 1000; i++) {
        const int *value = m.get("key" + std::to_string(i));
        ASSERT_NE(value, nullptr);
        EXPECT_EQ(*value, i);
    }
}

TEST(InsertionTest, JsonParseNestedObjectsWithParser) {
    struct Config {
        std::string name;
        int value;
        
        Config() = default;
        explicit Config(xjson::Parser &p):
            name(p.required<std::string>("name")),
            value(p.required<int>("value")) {}
    };
    
    const json j = {
        {"config1", {{"name", "first"}, {"value", 10}}},
        {"config2", {{"name", "second"}, {"value", 20}}},
        {"config3", {{"name", "third"}, {"value", 30}}}
    };
    
    xjson::Parser parser(j);
    map::Insertion<Config> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    
    const Config *cfg1 = m.get("config1");
    ASSERT_NE(cfg1, nullptr);
    EXPECT_EQ(cfg1->name, "first");
    EXPECT_EQ(cfg1->value, 10);
    
    const Config *cfg2 = m.get("config2");
    ASSERT_NE(cfg2, nullptr);
    EXPECT_EQ(cfg2->name, "second");
    EXPECT_EQ(cfg2->value, 20);
    
    const Config *cfg3 = m.get("config3");
    ASSERT_NE(cfg3, nullptr);
    EXPECT_EQ(cfg3->name, "third");
    EXPECT_EQ(cfg3->value, 30);
}

TEST(InsertionTest, JsonParseNestedObjectsWithParserError) {
    struct Config {
        std::string name;
        int value;
        
        Config() = default;
        explicit Config(xjson::Parser &p):
            name(p.required<std::string>("name")),
            value(p.required<int>("value")) {}
    };
    
    const json j = {
        {"config1", {{"name", "first"}, {"value", 10}}},
        {"config2", {{"name", "second"}}},  // Missing "value"
        {"config3", {{"name", "third"}, {"value", 30}}}
    };
    
    xjson::Parser parser(j);
    map::Insertion<Config> m(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_GE(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    EXPECT_EQ(err["path"], "config2.value");
    EXPECT_EQ(err["message"], "This field is required");
    // All configs should still be in the map
    EXPECT_EQ(m.count(), 3);
}

TEST(InsertionTest, JsonParseComplexNestedStructure) {
    struct Address {
        std::string street;
        std::string city;
        
        Address() = default;
        explicit Address(xjson::Parser &p):
            street(p.required<std::string>("street")),
            city(p.required<std::string>("city")) {}
    };
    
    struct Person {
        std::string name;
        int age;
        Address address;
        
        Person() = default;
        explicit Person(xjson::Parser &p):
            name(p.required<std::string>("name")),
            age(p.required<int>("age")) {
            auto addr_parser = p.child("address");
            address = Address(addr_parser);
        }
    };
    
    const json j = {
        {"person1", {
            {"name", "Alice"},
            {"age", 30},
            {"address", {{"street", "123 Main St"}, {"city", "New York"}}}
        }},
        {"person2", {
            {"name", "Bob"},
            {"age", 25},
            {"address", {{"street", "456 Elm St"}, {"city", "Boston"}}}
        }}
    };
    
    xjson::Parser parser(j);
    map::Insertion<Person> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 2);
    
    const Person *p1 = m.get("person1");
    ASSERT_NE(p1, nullptr);
    EXPECT_EQ(p1->name, "Alice");
    EXPECT_EQ(p1->age, 30);
    EXPECT_EQ(p1->address.street, "123 Main St");
    EXPECT_EQ(p1->address.city, "New York");
    
    const Person *p2 = m.get("person2");
    ASSERT_NE(p2, nullptr);
    EXPECT_EQ(p2->name, "Bob");
    EXPECT_EQ(p2->age, 25);
    EXPECT_EQ(p2->address.street, "456 Elm St");
    EXPECT_EQ(p2->address.city, "Boston");
}

TEST(InsertionTest, JsonParseMultipleTypesInOneObject) {
    const json j = {
        {"name", "test"},
        {"count", 42},
        {"enabled", true},
        {"score", 95.5}
    };
    
    // Parse as generic json values
    xjson::Parser parser(j);
    map::Insertion<json> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 4);
    
    EXPECT_EQ(m.get("name")->get<std::string>(), "test");
    EXPECT_EQ(m.get("count")->get<int>(), 42);
    EXPECT_EQ(m.get("enabled")->get<bool>(), true);
    EXPECT_NEAR(m.get("score")->get<double>(), 95.5, 0.001);
}

TEST(InsertionTest, JsonParseWithOptionalFields) {
    struct Config {
        std::string name;
        int value;
        bool enabled;
        
        Config() = default;
        explicit Config(xjson::Parser &p):
            name(p.required<std::string>("name")),
            value(p.optional<int>("value", 0)),
            enabled(p.optional<bool>("enabled", true)) {}
    };
    
    const json j = {
        {"cfg1", {{"name", "full"}, {"value", 100}, {"enabled", false}}},
        {"cfg2", {{"name", "partial"}, {"value", 50}}},
        {"cfg3", {{"name", "minimal"}}}
    };
    
    xjson::Parser parser(j);
    map::Insertion<Config> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    
    const Config *cfg1 = m.get("cfg1");
    EXPECT_EQ(cfg1->name, "full");
    EXPECT_EQ(cfg1->value, 100);
    EXPECT_FALSE(cfg1->enabled);
    
    const Config *cfg2 = m.get("cfg2");
    EXPECT_EQ(cfg2->name, "partial");
    EXPECT_EQ(cfg2->value, 50);
    EXPECT_TRUE(cfg2->enabled);  // Default value
    
    const Config *cfg3 = m.get("cfg3");
    EXPECT_EQ(cfg3->name, "minimal");
    EXPECT_EQ(cfg3->value, 0);  // Default value
    EXPECT_TRUE(cfg3->enabled);  // Default value
}

TEST(InsertionTest, JsonParseNullValues) {
    const json j = {
        {"valid", 42},
        {"null_value", nullptr}
    };
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    // Should accumulate an error for the null value
    EXPECT_FALSE(parser.ok());
    EXPECT_GE(parser.errors->size(), 1);
    // Valid entry should still be parsed
    EXPECT_EQ(m.count(), 1);
    EXPECT_EQ(*m.get("valid"), 42);
}

TEST(InsertionTest, JsonParseAllInvalidTypes) {
    const json j = {
        {"bad1", "not a number"},
        {"bad2", nullptr},
        {"bad3", json::array({1, 2, 3})}
    };
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_GE(parser.errors->size(), 3);
    EXPECT_TRUE(m.empty());
}

TEST(InsertionTest, JsonParseStringFromPrimitive) {
    const json j = {
        {"int_val", 42},
        {"float_val", 3.14},
        {"bool_val", true},
        {"string_val", "hello"}
    };
    xjson::Parser parser(j);
    map::Insertion<std::string> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 4);
    EXPECT_EQ(*m.get("int_val"), "42");
    EXPECT_EQ(*m.get("string_val"), "hello");
}

TEST(InsertionTest, JsonParseNestedErrorPropagation) {
    struct Inner {
        int required_field;
        
        Inner() = default;
        explicit Inner(xjson::Parser &p):
            required_field(p.required<int>("required_field")) {}
    };
    
    const json j = {
        {"obj1", {{"required_field", 10}}},
        {"obj2", {}},  // Missing required field
        {"obj3", {{"required_field", 30}}}
    };
    
    xjson::Parser parser(j);
    map::Insertion<Inner> m(parser);
    EXPECT_FALSE(parser.ok());
    EXPECT_EQ(parser.errors->size(), 1);
    auto err = parser.errors->at(0);
    // Error path should include parent key
    EXPECT_EQ(err["path"], "obj2.required_field");
}

TEST(InsertionTest, JsonParseSpecialCharacterKeys) {
    const json j = {
        {"key-with-dash", 1},
        {"key.with.dots", 2},
        {"key_with_underscore", 3},
        {"key with spaces", 4},
        {"key:colon", 5}
    };
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 5);
    EXPECT_EQ(*m.get("key-with-dash"), 1);
    EXPECT_EQ(*m.get("key.with.dots"), 2);
    EXPECT_EQ(*m.get("key_with_underscore"), 3);
    EXPECT_EQ(*m.get("key with spaces"), 4);
    EXPECT_EQ(*m.get("key:colon"), 5);
}

TEST(InsertionTest, JsonParseMixedValidAndInvalidNested) {
    struct Config {
        std::string name;
        int value;
        
        Config() = default;
        explicit Config(xjson::Parser &p):
            name(p.required<std::string>("name")),
            value(p.required<int>("value")) {}
    };
    
    const json j = {
        {"good1", {{"name", "first"}, {"value", 10}}},
        {"bad", {{"name", "missing_value"}}},
        {"good2", {{"name", "second"}, {"value", 20}}}
    };
    
    xjson::Parser parser(j);
    map::Insertion<Config> m(parser);
    EXPECT_FALSE(parser.ok());
    // Should have all three configs, but with errors for "bad"
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(m.get("good1")->name, "first");
    EXPECT_EQ(m.get("good1")->value, 10);
    EXPECT_EQ(m.get("good2")->name, "second");
    EXPECT_EQ(m.get("good2")->value, 20);
}

TEST(InsertionTest, JsonParseEmptyKey) {
    const json j = {
        {"", 42},
        {"normal_key", 100}
    };
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 2);
    EXPECT_EQ(*m.get(""), 42);
    EXPECT_EQ(*m.get("normal_key"), 100);
}

TEST(InsertionTest, JsonParseUnicodeKeys) {
    const json j = {
        {"français", 1},
        {"日本語", 2},
        {"עברית", 3}
    };
    xjson::Parser parser(j);
    map::Insertion<int> m(parser);
    EXPECT_TRUE(parser.ok());
    EXPECT_EQ(m.count(), 3);
    EXPECT_EQ(*m.get("français"), 1);
    EXPECT_EQ(*m.get("日本語"), 2);
    EXPECT_EQ(*m.get("עברית"), 3);
}
