//
// Created by Emiliano Bonilla on 11/16/25.
//

#pragma once
#include "arc/cpp/runtime/runtime.h"
#include "driver/pipeline/mock/pipeline.h"

namespace arc {
struct TaskConfig {};
class Source final: public pipeline::Source {
    std::shared_ptr<runtime::Runtime> runtime;

    explicit Source(const std::shared_ptr<runtime::Runtime> &runtime) : runtime(runtime) {}

    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &data) override {
        return this->runtime->read(data);
    }
    void stopped_with_err(const xerrors::Error &err) override {}
};

class Sink final: public pipeline::Sink {
    std::shared_ptr<runtime::Runtime> runtime;

    explicit Sink(const std::shared_ptr<runtime::Runtime> &runtime) : runtime(runtime) {}

    xerrors::Error write(telem::Frame &frame) override {
        return this->runtime->write(std::move(frame));
    }

};

class RuntimePipeline final: public pipeline::Base {
    std::shared_ptr<runtime::Runtime> runtime;

    explicit RuntimePipeline(
        const breaker::Config &breaker_config,
        const std::shared_ptr<runtime::Runtime> &runtime
        ): Base(breaker_config), runtime(runtime) {}

    void run() override {
        this->runtime->run();
    }
};

class TaskConfig final {
    std::string arc_key;
    arc::Arc


explicit TaskConfig(
    const std::shared_ptr<synnax::Synnax> &client,
    xjson::Parser &cfg
) {

}
};

class Task {
    pipeline::Acquisition source;
    pipeline::Control sink;
    std::shared_ptr<runtime::Runtime> runtime;
};
}