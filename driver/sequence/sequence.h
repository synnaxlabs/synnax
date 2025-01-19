#pragma once

#include "client/cpp/synnax.h"
#include "driver/loop/loop.h"
#include "lua.hpp"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

extern "C" {
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
}

struct LuaStateDeleter {
    void operator()(lua_State *L) const {
        if (L) lua_close(L);
    }
};

namespace sequence {

struct TaskConfig {
    synnax::Rate rate;
    std::vector<synnax::ChannelKey> read_from;
    std::vector<synnax::ChannelKey> write_to;

    explicit TaskConfig(config::Parser &parser): rate(synnax::Rate(
                                                     parser.required<float>("rate"))),
                                                 read_from(
                                                     parser.required_vector<
                                                         synnax::ChannelKey>(
                                                         "read_from")),
                                                 write_to(
                                                     parser.required_vector<
                                                         synnax::ChannelKey>(
                                                         "write_to")) {
    }
};


class Source {
public:
    virtual synnax::Frame read() = 0;

    virtual ~Source() = default;
};

class SynnaxSource : public pipeline::Sink, public Source {
public:
    synnax::Frame latest;

    synnax::Frame read() override {
        return latest.copy();
    }

    freighter::Error write(synnax::Frame frame) {
        this->latest = frame.copy();
        return freighter::Error();
    }
};

class Sink {
public:
    virtual freighter::Error write(synnax::Frame frame) = 0;

    virtual ~Sink() = default;
};

class SynnaxSink : public pipeline::Source, public Sink {
private:
    std::mutex mtx;
    std::condition_variable cv;
    synnax::Frame current_frame;
    bool frame_available = false;

public:
    std::pair<Frame, freighter::Error> read(breaker::Breaker &breaker) override {
        std::unique_lock<std::mutex> lock(mtx);
        cv.wait(lock, [this]() { return frame_available; });

        frame_available = false;
        return {current_frame.copy(), freighter::Error()};
    }

    freighter::Error write(synnax::Frame frame) override {
        std::lock_guard<std::mutex> lock(mtx);
        current_frame = frame.copy();
        frame_available = true;
        cv.notify_one();
        return freighter::Error();
    }
};

class Sequence {
private:
    /// @brief loop is used to regulate the execution speed of the sequence
    loop::Timer loop;
    /// @brief source is used to read channel values.
    std::shared_ptr<Source> source;
    /// @brief sink is used to write channel values.
    std::shared_ptr<Sink> sink;
    /// @brief L is the Lua state used to execute the sequence.
    std::unique_ptr<lua_State, LuaStateDeleter> L;
    /// @brief channels is the map of channel keys to channel information.
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    /// @brief latest_values stores the most recent value received for each channel
    std::unordered_map<synnax::ChannelKey, synnax::Series> latest_values;
    synnax::Frame output;
    std::string script;
    /// @brief Lua callback function to set values in the output frame
    /// @param L The Lua state
    int lua_set(lua_State *L) {
        // Check arguments: (channel_name, value)
        if (lua_gettop(L) != 2) {
            luaL_error(
                L,
                "set() requires 2 arguments: channel_name (string) and value (number)");
            return 0;
        }

        // Get channel name
        if (!lua_isstring(L, 1)) {
            luaL_error(L, "First argument must be a string (channel name)");
            return 0;
        }
        std::string channel_name = lua_tostring(L, 1);

        // Get value
        if (!lua_isnumber(L, 2)) {
            luaL_error(L, "Second argument must be a number");
            return 0;
        }
        double value = lua_tonumber(L, 2);

        // Find the channel key by name
        synnax::ChannelKey target_key;
        bool found = false;
        const synnax::Channel* target_channel = nullptr;
        for (const auto &[key, channel]: channels) {
            if (channel.name == channel_name) {
                target_key = key;
                target_channel = &channel;
                found = true;
                break;
            }
        }

        if (!found) {
            luaL_error(L, ("Channel not found: " + channel_name).c_str());
            return 0;
        }

        // Create a series with the correct data type
        if (target_channel->data_type == synnax::FLOAT64) {
            this->output.add(target_key, synnax::Series(static_cast<double>(value)));
        } else if (target_channel->data_type == synnax::FLOAT32) {
            this->output.add(target_key, synnax::Series(static_cast<float>(value)));
        } else if (target_channel->data_type == synnax::INT64) {
            this->output.add(target_key, synnax::Series(static_cast<int64_t>(value)));
        } else if (target_channel->data_type == synnax::INT32) {
            this->output.add(target_key, synnax::Series(static_cast<int32_t>(value)));
        } else if (target_channel->data_type == synnax::INT16) {
            this->output.add(target_key, synnax::Series(static_cast<int16_t>(value)));
        } else if (target_channel->data_type == synnax::INT8) {
            this->output.add(target_key, synnax::Series(static_cast<int8_t>(value)));
        } else if (target_channel->data_type == synnax::UINT64) {
            this->output.add(target_key, synnax::Series(static_cast<uint64_t>(value)));
        } else if (target_channel->data_type == synnax::UINT32) {
            this->output.add(target_key, synnax::Series(static_cast<uint32_t>(value)));
        } else if (target_channel->data_type == synnax::SY_UINT16) {
            this->output.add(target_key, synnax::Series(static_cast<uint16_t>(value)));
        } else if (target_channel->data_type == synnax::SY_UINT8) {
            this->output.add(target_key, synnax::Series(static_cast<uint8_t>(value)));
        } else {
            luaL_error(L, ("Unsupported data type for channel: " + channel_name).c_str());
            return 0;
        }

        return 0;
    }

    /// @brief Static wrapper for lua_set that retrieves the Sequence instance
    static int lua_set_wrapper(lua_State *L) {
        // Get the Sequence instance from upvalue
        Sequence *seq = static_cast<Sequence *>(lua_touserdata(L, lua_upvalueindex(1)));
        return seq->lua_set(L);
    }

    /// @brief Registers the set function in the Lua environment
    void registerSetFunction() {
        // Push the sequence instance as userdata
        lua_pushlightuserdata(L.get(), this);

        // Push the static wrapper function with the sequence instance as upvalue
        lua_pushcclosure(L.get(), &Sequence::lua_set_wrapper, 1);

        // Register it as a global function named 'set'
        lua_setglobal(L.get(), "set");
    }

public:
    Sequence(
        synnax::Rate rate,
        std::shared_ptr<Source> source,
        std::shared_ptr<Sink> sink,
        std::unordered_map<synnax::ChannelKey, synnax::Channel> channels,
        std::string script
    ): loop(rate), source(source), sink(sink), channels(channels), script(script) {
        L.reset(luaL_newstate());
        luaL_openlibs(L.get());
        registerSetFunction(); // Register the set function
    }

    void main(breaker::Breaker &breaker) {
        while (breaker.running()) {
            // Run forever if iters <= 0, otherwise run for specified iterations
            // Read the current frame from the source
            auto frame = source->read();

            // Set frame variables in Lua environment
            setFrameVariables(frame);

            // Reset output frame for new values
            output = synnax::Frame(channels.size());

            // Execute the Lua script
            if (luaL_dostring(L.get(), script.c_str()) != LUA_OK) {
                const char *error_msg = lua_tostring(L.get(), -1);
                // TODO: Handle Lua execution error
                break;
            }

            // Write the output frame to the sink
            auto err = sink->write(std::move(output));
            if (err) {
                // Handle error
                break;
            }
        }
    }

    /// @brief Sets the frame data as Lua variables using channel names
    /// @param frame The frame containing channel data to expose to Lua
    void setFrameVariables(const synnax::Frame &frame) {
        if (frame.channels != nullptr) {
            for (size_t i = 0; i < frame.channels->size(); i++) {
                const auto &channel_key = frame.channels->at(i);
                const auto &series = frame.series->at(i);

                // Create a temporary Series and move it into the map
                if (series.data_type == synnax::FLOAT64) {
                    double val = series.at<double>(0);
                    this->latest_values.insert_or_assign(
                        channel_key, synnax::Series(val, synnax::FLOAT64));
                } else if (series.data_type == synnax::FLOAT32) {
                    float val = series.at<float>(0);
                    this->latest_values.insert_or_assign(
                        channel_key, synnax::Series(val, synnax::FLOAT32));
                } else if (series.data_type == synnax::INT64) {
                    int64_t val = series.at<int64_t>(0);
                    this->latest_values.insert_or_assign(
                        channel_key, synnax::Series(val, synnax::INT64));
                } else if (series.data_type == synnax::INT32) {
                    int32_t val = series.at<int32_t>(0);
                    this->latest_values.insert_or_assign(
                        channel_key, synnax::Series(val, synnax::INT32));
                } else {
                    this->latest_values.insert_or_assign(
                        channel_key, synnax::Series(0.0, synnax::FLOAT64));
                }
            }
        }

        // Now iterate through all channels and set their Lua variables
        for (const auto &[key, channel]: this->channels) {
            const auto &var_name = channel.name;
            lua_pushstring(this->L.get(), var_name.c_str());

            // Try to get value from latest_values cache
            auto it = this->latest_values.find(key);
            if (it != this->latest_values.end()) {
                const auto &series = it->second;
                if (series.data_type == synnax::FLOAT64)
                    lua_pushnumber(this->L.get(), series.at<double>(0));
                else if (series.data_type == synnax::FLOAT32)
                    lua_pushnumber(this->L.get(), series.at<float>(0));
                else if (series.data_type == synnax::INT64)
                    lua_pushinteger(this->L.get(), series.at<int64_t>(0));
                else if (series.data_type == synnax::INT32)
                    lua_pushinteger(this->L.get(), series.at<int32_t>(0));
                else
                    lua_pushnumber(this->L.get(), 0.0); // Default for unsupported types
            } else {
                // No value has been received yet, use default value of 0
                lua_pushnumber(this->L.get(), 0.0);
            }

            lua_setglobal(this->L.get(), var_name.c_str());
        }
    }
};

class Task final : public task::Task {
public:
    std::atomic<bool> running = false;
    pipeline::Control input;
    pipeline::Acquisition output;
    std::unique_ptr<Sequence> sequence;
    std::unique_ptr<breaker::Breaker> breaker;
    std::thread sequence_thread;

    Task(
        const std::shared_ptr<task::Context> &ctx,
        synnax::WriterConfig writer_config,
        synnax::StreamerConfig streamer_config,
        std::shared_ptr<pipeline::Sink> p_sink,
        std::shared_ptr<pipeline::Source> p_source,
        std::shared_ptr<Source> source,
        std::shared_ptr<Sink> sink,
        std::unique_ptr<Sequence> seq,
        const breaker::Config breaker_config
    ): output(
           pipeline::Acquisition(
               ctx->client,
               writer_config,
               p_source,
               breaker_config
           )
       ),
       input(
           pipeline::Control(
               ctx->client,
               streamer_config,
               p_sink,
               breaker_config
           )
       ),
       sequence(std::move(seq)),
       breaker(std::make_unique<breaker::Breaker>(breaker_config))
    {
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
    }

    void start(const std::string &key) {
        if (this->running.exchange(true)) return;
        this->breaker->reset();
        this->output.start();
        this->input.start();
        this->breaker->start();

        this->sequence_thread = std::thread([this]() {
            this->sequence->main(*this->breaker);
        });
    }

    void stop() override { this->stop(""); }

    void stop(const std::string &key) {
        if (!this->running.exchange(false)) return;
        this->breaker->stop();
        if (this->sequence_thread.joinable()) {
            this->sequence_thread.join();
        }
        this->output.stop();
        this->input.stop();
    }

    static std::unique_ptr<task::Task> configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    LOG(INFO) << "[sequence] configuring task " << task.name;
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    auto parser = config::Parser(task.config);
    TaskConfig config(parser);
    auto control_subject = synnax::ControlSubject{
        .name = task.name,
        .key = task.name + "-" + std::to_string(task.key)
    };

    // Create sources and sinks
    auto synnax_source = std::make_shared<SynnaxSource>();
    auto synnax_sink = std::make_shared<SynnaxSink>();

    auto writer_config = synnax::WriterConfig{
        .channels = config.write_to,
        .subject = control_subject,
        .mode = synnax::WriterMode::PersistStream,
        .enable_auto_commit = true,
    };

    auto streamer_config = synnax::StreamerConfig{
        .channels = config.read_from,
    };

    // Create channel map
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    for (const auto& key : config.read_from) {
        auto [ch, err] = ctx->client->channels.retrieve(key);
        if (err) {
            // Handle error
        }
    }
    for (const auto& key : config.write_to) {
        auto [ch, err] = ctx->client->channels.retrieve(key);
        if (err) {
            // Handle error
        }
        channels[key] = ch;
    }

    // Create sequence
    auto sequence = std::make_unique<Sequence>(
        config.rate,
        synnax_source,
        synnax_sink,
        channels,
        parser.required<std::string>("script")
    );

    return std::make_unique<Task>(
        ctx,
        writer_config,
        streamer_config,
        synnax_source,
        synnax_sink,
        synnax_source,
        synnax_sink,
        std::move(sequence),
        breaker_config
    );
}
};



class Factory final: public task::Factory {
public:
    Factory() = default;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type != "sequence") return {nullptr, false};
        return {sequence::Task::configure(ctx, task), true};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        return {};
    }
};

};
