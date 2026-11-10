// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <set>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): one seed per suite,
// printed so that a failing run can be replayed.
std::mt19937 gen_rand_device = random_generator("Device Tests");
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

namespace synnax::device {
/// @brief it should correctly create a device.
TEST(DeviceTests, testCreateDevice) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto d = Device{
        .key = "asdfjahsdfkasjdfhaks_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = {{"test", "properties"}}
    };
    ASSERT_NIL(client.devices.create(d));
    ASSERT_EQ(d.name, "test_device");
}

/// @brief it should correctly retrieve a device.
TEST(DeviceTests, testRetrieveDevice) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto d = Device{
        .key = "asdfjahsdfkasjdfhaks_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = {{"test", "properties"}}
    };
    ASSERT_NIL(client.devices.create(d));
    const auto d2 = ASSERT_NIL_P(client.devices.retrieve(d.key));
    ASSERT_EQ(d2.name, "test_device");
    ASSERT_EQ(d2.key, d.key);
}

/// @brief it should set the fields of a device with a dispatch, and keep the others.
TEST(DeviceTests, testDispatch) {
    const auto client = new_test_client();
    const auto r = ASSERT_NIL_P(client.racks.create("test_rack"));
    auto d = Device{
        .key = "dispatch_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "slot-1",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = {{"a", "1"}},
    };
    ASSERT_NIL(client.devices.create(d));
    const x::json::json properties = {{"a", "2"}, {"b", "3"}};
    ASSERT_NIL(client.devices.dispatch(
        d.key,
        {set({.path = {"location"}, .value = "slot-2"}),
         set({.path = {"model"}, .value = "other_model"}),
         set({.path = {"properties"}, .value = properties})}
    ));
    const auto got = ASSERT_NIL_P(client.devices.retrieve(d.key));
    ASSERT_EQ(got.location, "slot-2");
    ASSERT_EQ(got.model, "other_model");
    ASSERT_EQ(x::json::json(got.properties), properties);
    ASSERT_EQ(got.name, "test_device");
    ASSERT_EQ(got.make, "test_make");
}

/// @brief it should move a device to another rack and parent with a dispatch.
TEST(DeviceTests, testDispatchSetRack) {
    const auto client = new_test_client();
    const auto from = ASSERT_NIL_P(client.racks.create("test_rack_from"));
    const auto to = ASSERT_NIL_P(client.racks.create("test_rack_to"));
    const auto rand = std::to_string(gen_rand_device());
    auto chassis = Device{
        .key = "dispatch_chassis_" + rand,
        .rack = to.key,
        .location = "chassis",
        .make = "test_make",
        .model = "test_model",
        .name = "chassis",
    };
    ASSERT_NIL(client.devices.create(chassis));
    auto d = Device{
        .key = "dispatch_module_" + rand,
        .rack = from.key,
        .location = "slot-1",
        .make = "test_make",
        .model = "test_model",
        .name = "module",
    };
    ASSERT_NIL(client.devices.create(d));
    ASSERT_NIL(client.devices.dispatch(
        d.key,
        {set_rack({.rack = to.key, .parent = chassis.key})}
    ));
    const auto got = ASSERT_NIL_P(client.devices.retrieve(d.key));
    ASSERT_EQ(got.rack, to.key);
    ASSERT_EQ(got.parent, chassis.key);
}

/// @brief it should return a not found error when it dispatches to a device that does
/// not exist.
TEST(DeviceTests, testDispatchNotFound) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS(
        client.devices.dispatch(
            "missing_" + std::to_string(gen_rand_device()),
            {set({.path = {"name"}, .value = "renamed"})}
        ),
        x::errors::NOT_FOUND
    );
}

/// @brief it should correctly retrieve multiple devices.
TEST(DeviceTests, testRetrieveDevices) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "test_device_1",
        .properties = {{"key", "properties_1"}}
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "test_device_2",
        .properties = {{"key", "properties_2"}}
    };
    ASSERT_NIL(client.devices.create(d2));

    const std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(keys));

    ASSERT_EQ(devices.size(), 2);

    const auto it1 = std::ranges::find_if(devices, [&d1](const Device &d) {
        return d.key == d1.key;
    });
    ASSERT_NE(it1, devices.end());
    ASSERT_EQ(it1->name, "test_device_1");
    ASSERT_EQ(it1->location, "location_1");

    auto it2 = std::ranges::find_if(devices, [&d2](const Device &d) {
        return d.key == d2.key;
    });
    ASSERT_NE(it2, devices.end());
    ASSERT_EQ(it2->name, "test_device_2");
    ASSERT_EQ(it2->location, "location_2");
}

/// @brief it should correctly create multiple devices at once.
TEST(DeviceTests, testCreateDevices) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    std::vector devices = {
        Device{
            .key = "device1_key_" + std::to_string(gen_rand_device()),
            .rack = r.key,
            .location = "location_1",
            .make = "make_1",
            .model = "model_1",
            .name = "test_device_1",
            .properties = {{"properties_1", "value"}},
        },
        Device{
            .key = "device2_key_" + std::to_string(gen_rand_device()),
            .rack = r.key,
            .location = "location_2",
            .make = "make_2",
            .model = "model_2",
            .name = "test_device_2",
            .properties = {{"properties_2", "value"}},
        },
        Device{
            .key = "device3_key_" + std::to_string(gen_rand_device()),
            .rack = r.key,
            .location = "location_3",
            .make = "make_3",
            .model = "model_3",
            .name = "test_device_3",
            .properties = {{"properties_3", "value3"}},
        },
    };

    ASSERT_NIL(client.devices.create(devices));

    for (const auto &device: devices) {
        const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(device.key));
        ASSERT_EQ(retrieved.key, device.key);
        ASSERT_EQ(retrieved.name, device.name);
        ASSERT_EQ(retrieved.rack, r.key);
        ASSERT_EQ(retrieved.location, device.location);
        ASSERT_EQ(retrieved.make, device.make);
        ASSERT_EQ(retrieved.model, device.model);
        ASSERT_EQ(retrieved.properties, device.properties);
    }

    std::vector<std::string> keys;
    keys.reserve(devices.size());
    for (const auto &device: devices)
        keys.push_back(device.key);

    const auto retrieved_devices = ASSERT_NIL_P(client.devices.retrieve(keys));
    ASSERT_EQ(retrieved_devices.size(), devices.size());

    auto device_map = map_device_keys(retrieved_devices);

    for (const auto &device: devices) {
        ASSERT_TRUE(device_map.contains(device.key));
        const auto &retrieved = device_map[device.key];
        ASSERT_EQ(retrieved.name, device.name);
        ASSERT_EQ(retrieved.rack, r.key);
    }
}

/// @brief it should correctly handle the configured field.
TEST(DeviceTests, testDeviceConfigured) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "device_1",
        .configured = false,
        .properties = {{"key", "properties_1"}},
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "device_2",
        .configured = true,
        .properties = {{"key", "properties_2"}},
    };
    ASSERT_NIL(client.devices.create(d2));

    const auto retrieved1 = ASSERT_NIL_P(client.devices.retrieve(d1.key));
    ASSERT_FALSE(retrieved1.configured);

    const auto retrieved2 = ASSERT_NIL_P(client.devices.retrieve(d2.key));
    ASSERT_TRUE(retrieved2.configured);

    const std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(keys));
    auto device_map = map_device_keys(devices);

    ASSERT_FALSE(device_map[d1.key].configured);
    ASSERT_TRUE(device_map[d2.key].configured);
}

/// @brief it should correctly handle retrieving devices after deletion.
TEST(DeviceTests, testRetrieveDevicesAfterDeletion) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "test_device_1",
        .properties = {{"key", "properties_1"}}
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "test_device_2",
        .properties = {{"key", "properties_2"}}
    };
    ASSERT_NIL(client.devices.create(d2));

    ASSERT_NIL(client.devices.del(d1.key));

    ASSERT_OCCURRED_AS_P(client.devices.retrieve(d1.key), x::errors::NOT_FOUND);

    const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(d2.key));
    ASSERT_EQ(retrieved.key, d2.key);
    ASSERT_EQ(retrieved.name, "test_device_2");
}

/// @brief it should correctly delete a device.
TEST(DeviceTests, testDeleteDevice) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d = Device{
        .key = "device_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = {{"test", "properties"}}
    };
    ASSERT_NIL(client.devices.create(d));
    ASSERT_NIL(client.devices.del(d.key));

    ASSERT_OCCURRED_AS_P(client.devices.retrieve(d.key), x::errors::NOT_FOUND);
}

/// @brief it should correctly delete multiple devices.
TEST(DeviceTests, testDeleteDevices) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "device_1",
        .properties = {{"key", "properties_1"}}
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "device_2",
        .properties = {{"key", "properties_2"}}
    };
    ASSERT_NIL(client.devices.create(d2));

    const std::vector keys = {d1.key, d2.key};
    ASSERT_NIL(client.devices.del(keys));

    ASSERT_TRUE(ASSERT_NIL_P(client.devices.retrieve(keys)).empty());
}

/// @brief it should retrieve devices with a Where on keys and a Where on names.
TEST(DeviceTests, testRetrieveWithRequest) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d1 = Device{
        .key = "req_d1_" + rand,
        .rack = r.key,
        .location = "loc_a",
        .make = "make_a",
        .model = "model_a",
        .name = "req_device_1_" + rand,
        .properties = {{"key", "p1"}}
    };
    auto d2 = Device{
        .key = "req_d2_" + rand,
        .rack = r.key,
        .location = "loc_b",
        .make = "make_b",
        .model = "model_b",
        .name = "req_device_2_" + rand,
        .properties = {{"key", "p2"}}
    };
    auto d3 = Device{
        .key = "req_d3_" + rand,
        .rack = r.key,
        .location = "loc_c",
        .make = "make_c",
        .model = "model_c",
        .name = "req_device_3_" + rand,
        .properties = {{"key", "p3"}}
    };
    ASSERT_NIL(client.devices.create(d1));
    ASSERT_NIL(client.devices.create(d2));
    ASSERT_NIL(client.devices.create(d3));
    const auto devices_keys = ASSERT_NIL_P(
        client.devices.retrieve(Where{.key = {d1.key, d3.key}})
    );
    ASSERT_EQ(devices_keys.size(), 2);
    auto dm = map_device_keys(devices_keys);
    ASSERT_TRUE(dm.contains(d1.key));
    ASSERT_TRUE(dm.contains(d3.key));
    const auto devices_names = ASSERT_NIL_P(
        client.devices.retrieve(Where{.name = {d1.name, d2.name}})
    );
    ASSERT_EQ(devices_names.size(), 2);
}

/// @brief it should retrieve the devices that match any branch of an or.
TEST(DeviceTests, testRetrieveWhere) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    std::vector<Device> devs;
    for (int i = 0; i < 3; ++i) {
        auto d = Device{
            .key = "where_d" + std::to_string(i) + "_" + rand,
            .rack = r.key,
            .location = "loc",
            .make = "where_make_" + std::to_string(i),
            .model = "model",
            .name = "where_device_" + std::to_string(i),
        };
        ASSERT_NIL(client.devices.create(d));
        devs.push_back(d);
    }
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(
        (Where{.make = devs[0].make} || Where{.key = devs[1].key}) &&
        Where{.rack = r.key}
    ));
    std::set<Key> keys;
    for (const auto &d: devices)
        keys.insert(d.key);
    ASSERT_EQ(keys, (std::set<Key>{devs[0].key, devs[1].key}));
}

/// @brief it should treat a missing key as a non-match in a list retrieve.
TEST(DeviceTests, testRetrieveMissingKey) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto d = Device{
        .key = "missing_d_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "loc",
        .make = "make",
        .model = "model",
        .name = "missing_device",
    };
    ASSERT_NIL(client.devices.create(d));
    const auto devices = ASSERT_NIL_P(
        client.devices.retrieve(std::vector<std::string>{d.key, "missing_d_none"})
    );
    ASSERT_EQ(devices.size(), 1);
    ASSERT_EQ(devices[0].key, d.key);
}
/// @brief it should page a retrieve of devices by filter or by keys.
TEST(DeviceTests, testRetrieveWithLimitAndCursor) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    const auto make = "limit_make_" + rand;
    std::vector<Device> devices;
    for (int i = 0; i < 5; ++i) {
        auto d = Device{
            .key = "limit_d_" + rand + "_" + std::to_string(i),
            .rack = r.key,
            .location = "loc",
            .make = make,
            .model = "model",
            .name = "limit_device_" + std::to_string(i),
            .properties = {{"key", "props"}}
        };
        ASSERT_NIL(client.devices.create(d));
        devices.push_back(d);
    }
    const auto where = Where{.make = make};
    const auto devices_limited = ASSERT_NIL_P(
        client.devices.retrieve(where, {.limit = 2})
    );
    ASSERT_EQ(devices_limited.size(), 2);
    const auto devices_after = ASSERT_NIL_P(client.devices.retrieve(
        where,
        {.after = Cursor{.key = devices_limited[1].key}, .limit = 2}
    ));
    ASSERT_EQ(devices_after.size(), 2);
    bool different = true;
    for (const auto &da: devices_limited)
        for (const auto &db: devices_after)
            if (da.key == db.key) different = false;
    ASSERT_TRUE(different);
    std::vector<std::string> keys;
    keys.reserve(devices.size());
    for (const auto &d: devices)
        keys.push_back(d.key);
    const auto by_key = ASSERT_NIL_P(client.devices.retrieve(keys, {.limit = 2}));
    ASSERT_EQ(by_key.size(), 2);
    const auto by_key_rest = ASSERT_NIL_P(
        client.devices.retrieve(keys, {.after = Cursor{.key = keys[3]}})
    );
    ASSERT_EQ(by_key_rest.size(), 1);
}

/// @brief it should reject a filter for one device that more than one device matches.
TEST(DeviceTests, testRetrieveOneWhereMultipleFound) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto make = "one_make_" + std::to_string(gen_rand_device());
    for (int i = 0; i < 2; ++i) {
        auto d = Device{
            .key = make + "_" + std::to_string(i),
            .rack = r.key,
            .location = "loc",
            .make = make,
            .model = "model",
            .name = "one_device_" + std::to_string(i),
        };
        ASSERT_NIL(client.devices.create(d));
    }
    ASSERT_OCCURRED_AS_P(
        client.devices.retrieve(Where{.make = make}, x::filter::one),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should retrieve the one device that a filter matches.
TEST(DeviceTests, testRetrieveOneWhere) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto d = Device{
        .key = "one_d_" + std::to_string(gen_rand_device()),
        .rack = r.key,
        .location = "loc",
        .make = "make",
        .model = "model",
        .name = "one_device",
    };
    ASSERT_NIL(client.devices.create(d));
    const auto got = ASSERT_NIL_P(
        client.devices.retrieve(Where{.key = d.key}, x::filter::one)
    );
    ASSERT_EQ(got.key, d.key);
}

/// @brief it should name the filter in the error when no device matches it.
TEST(DeviceTests, testRetrieveOneWhereNotFound) {
    const auto client = new_test_client();
    const auto make = "none_make_" + std::to_string(gen_rand_device());
    const auto [_, err] = client.devices.retrieve(Where{.make = make}, x::filter::one);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, "Found no device where make = \"" + make + "\"");
}

/// @brief it should reject a limit or a cursor on a retrieve of one device.
TEST(DeviceTests, testRetrieveDeviceRejectsPage) {
    const auto client = new_test_client();
    for (const auto &options:
         {RetrieveOptions{.limit = 1}, RetrieveOptions{.after = Cursor{}}})
        for (const auto &err:
             {client.devices.retrieve("device", options).second,
              client.devices.retrieve(Where{.key = "device"}, x::filter::one, options)
                  .second}) {
            ASSERT_MATCHES(err, x::errors::VALIDATION);
            ASSERT_EQ(
                err.data,
                "a query for one record takes no limit or cursor, since a page could "
                "hide a second match"
            );
        }
}

/// @brief it should correctly create and retrieve a device with a status.
TEST(DeviceTests, testCreateDeviceWithStatus) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d = Device{
        .key = "status_dev_" + rand,
        .rack = r.key,
        .location = "location",
        .make = "make",
        .model = "model",
        .name = "device_with_status",
        .properties = {{"key", "properties"}}
    };
    d.status = device::Status{
        .name = "Device status",
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Device is connected",
        .time = x::telem::TimeStamp::now(),
        .details = device::StatusDetails{.rack = r.key}
    };
    ASSERT_NIL(client.devices.create(d));
    const auto d2 = ASSERT_NIL_P(
        client.devices.retrieve(d.key, {.include_status = true})
    );
    ASSERT_EQ(d2.name, "device_with_status");
    ASSERT_TRUE(d.status.has_value());
    ASSERT_EQ(d2.status.value().variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_EQ(d2.status.value().message, "Device is connected");
    ASSERT_EQ(d2.status.value().details.rack, r.key);
}

/// @brief it should correctly retrieve multiple devices with statuses.
TEST(DeviceTests, testRetrieveDevicesWithStatus) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d1 = Device{
        .key = "status_d1_" + rand,
        .rack = r.key,
        .location = "loc1",
        .make = "make1",
        .model = "model1",
        .name = "device_1_status",
        .properties = {{"key", "props1"}},
        .status = device::Status{
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Device 1 OK",
            .time = x::telem::TimeStamp::now(),
        }
    };
    auto d2 = Device{
        .key = "status_d2_" + rand,
        .rack = r.key,
        .location = "loc2",
        .make = "make2",
        .model = "model2",
        .name = "device_2_status",
        .properties = {{"key", "props2"}},
        .status = device::Status{
            .variant = synnax::status::VARIANT_WARNING,
            .message = "Device 2 Warning",
            .time = x::telem::TimeStamp::now(),
        }
    };
    ASSERT_NIL(client.devices.create(d1));
    ASSERT_NIL(client.devices.create(d2));
    const std::vector<std::string> keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(
        client.devices.retrieve(keys, {.include_status = true})
    );
    ASSERT_EQ(devices.size(), 2);
    auto dm = map_device_keys(devices);
    ASSERT_EQ(dm[d1.key].status.value().variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_EQ(dm[d1.key].status.value().message, "Device 1 OK");
    ASSERT_EQ(dm[d2.key].status.value().variant, synnax::status::VARIANT_WARNING);
    ASSERT_EQ(dm[d2.key].status.value().message, "Device 2 Warning");
}

/// @brief it should correctly parse device::StatusDetails from JSON.
TEST(DeviceStatusDetailsTests, testParseFromJSON) {
    x::json::json j = {{"rack", 12345}};
    const x::json::Parser parser(j);
    const auto details = device::StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(details.rack, 12345);
}

/// @brief it should correctly serialize device::StatusDetails to JSON.
TEST(DeviceStatusDetailsTests, testToJSON) {
    const device::StatusDetails details{.rack = 67890};
    const auto j = details.to_json();
    ASSERT_EQ(j["rack"], 67890);
}

/// @brief it should round-trip device::StatusDetails through JSON.
TEST(DeviceStatusDetailsTests, testRoundTrip) {
    device::StatusDetails original{.rack = 11111};
    const auto j = original.to_json();
    x::json::Parser parser(j);
    auto recovered = device::StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(recovered.rack, original.rack);
}

/// @brief it should correctly parse a Device from JSON.
TEST(DeviceTests, testParseFromJSON) {
    x::json::json j = {
        {"key", "json-device-key"},
        {"name", "json-device-name"},
        {"rack", 99999},
        {"location", "json-location"},
        {"make", "json-make"},
        {"model", "json-model"},
        {"properties", {{"custom", true}}},
        {"configured", true}
    };
    x::json::Parser parser(j);
    auto d = Device::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(d.key, "json-device-key");
    ASSERT_EQ(d.name, "json-device-name");
    ASSERT_EQ(d.rack, 99999);
    ASSERT_EQ(d.make, "json-make");
    ASSERT_EQ(d.model, "json-model");
    ASSERT_EQ(x::json::json(d.properties), x::json::json({{"custom", true}}));
    ASSERT_EQ(d.configured, true);
}

/// @brief it should handle null properties when parsing Device from JSON.
TEST(DeviceTests, testParseFromJSONNullProperties) {
    x::json::json j = {
        {"key", "null-props-device"},
        {"name", "device-with-null-props"},
        {"rack", 1},
        {"location", "loc"},
        {"make", "make"},
        {"model", "model"},
        {"properties", nullptr},
        {"configured", false}
    };
    x::json::Parser parser(j);
    auto d = Device::parse(parser);
    ASSERT_EQ(d.key, "null-props-device");
    ASSERT_EQ(x::json::json(d.properties), x::json::json::object());
}

/// @brief it should handle default values when parsing Device from JSON.
TEST(DeviceTests, testParseFromJSONDefaults) {
    const x::json::json j = x::json::json::object();
    x::json::Parser parser(j);
    const auto d = Device::parse(parser);
    ASSERT_EQ(d.key, "");
    ASSERT_EQ(d.name, "");
    ASSERT_EQ(d.rack, 0);
    ASSERT_EQ(d.make, "");
    ASSERT_EQ(d.model, "");
    ASSERT_EQ(d.configured, false);
}

/// @brief it should create a device held by another device.
TEST(DeviceTests, testCreateWithParent) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto chassis = Device{
        .key = "chassis_" + rand,
        .rack = r.key,
        .location = "slot_0",
        .make = "NI",
        .model = "cDAQ-9178",
        .name = "chassis_" + rand,
        .properties = {{"is_chassis", true}},
    };
    ASSERT_NIL(client.devices.create(chassis));
    auto module = Device{
        .key = "module_" + rand,
        .rack = r.key,
        .location = "slot_1",
        .make = "NI",
        .model = "NI-9205",
        .name = "module_" + rand,
        .properties = {{"is_chassis", false}},
        .parent = chassis.key,
    };
    ASSERT_NIL(client.devices.create(module));
    const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(module.key));
    ASSERT_EQ(retrieved.key, module.key);
    ASSERT_EQ(retrieved.name, module.name);
    EXPECT_EQ(retrieved.parent, chassis.key);
}

TEST(DeviceTests, testCreateMultipleDevicesWithParent) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto chassis = Device{
        .key = "chassis_multi_" + rand,
        .rack = r.key,
        .location = "slot_0",
        .make = "NI",
        .model = "cDAQ-9178",
        .name = "chassis_multi_" + rand,
        .properties = {{"is_chassis", true}},
    };
    ASSERT_NIL(client.devices.create(chassis));
    auto mod1 = Device{
        .key = "mod1_" + rand,
        .rack = r.key,
        .location = "slot_1",
        .make = "NI",
        .model = "NI-9205",
        .name = "mod1_" + rand,
        .properties = {{"is_chassis", false}},
        .parent = chassis.key,
    };
    auto mod2 = Device{
        .key = "mod2_" + rand,
        .rack = r.key,
        .location = "slot_2",
        .make = "NI",
        .model = "NI-9264",
        .name = "mod2_" + rand,
        .properties = {{"is_chassis", false}},
        .parent = chassis.key,
    };
    ASSERT_NIL(client.devices.create(mod1));
    ASSERT_NIL(client.devices.create(mod2));
    const auto retrieved = ASSERT_NIL_P(
        client.devices.retrieve(std::vector<std::string>{mod1.key, mod2.key})
    );
    ASSERT_EQ(retrieved.size(), 2);
}

/// @brief it should refuse a parent device that does not exist.
TEST(DeviceTests, testCreateDeviceWithMissingParent) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto child = Device{
        .key = "orphan_" + rand,
        .rack = r.key,
        .location = "slot_1",
        .make = "NI",
        .model = "NI-9205",
        .name = "orphan_" + rand,
        .parent = "missing_" + rand,
    };
    const auto err = client.devices.create(child);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_TRUE(err.data.contains(
        "device missing_" + rand + " does not exist: validation error"
    )) << err;
    EXPECT_TRUE(err.data.contains(R"("path":["parent"])")) << err;
}

/// @brief it should create a device with no parent when none is given.
TEST(DeviceTests, testCreateDeviceWithNoParent) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d = Device{
        .key = "no_parent_" + rand,
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "no_parent_device_" + rand,
        .properties = {{"test", "properties"}},
    };
    ASSERT_NIL(client.devices.create(d));
    const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(d.key));
    ASSERT_EQ(retrieved.key, d.key);
    EXPECT_FALSE(retrieved.parent.has_value());
}
}
