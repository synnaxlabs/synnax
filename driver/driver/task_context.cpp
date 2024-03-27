//
// Created by Emiliano Bonilla on 3/27/24.
//

#include <glog/logging.h>
#include "nlohmann/json.hpp"

/// Internal.
#include "driver/driver/driver.h"

using json = nlohmann::json;

driver::TaskContext::TaskContext(const std::shared_ptr<Synnax>& client) : client(client) {
}


const std::string TASK_STATE_CHANNEL = "sy_task_state";

void driver::TaskContext::setState(TaskState state) {
    state_mutex.lock();
    if (state_updater == nullptr) {
        auto [task_state_ch, err] = client->channels.retrieve(TASK_STATE_CHANNEL);
        if (err) {
            LOG(ERROR) << "Failed to retrieve channel to update task state" << err.message();
            state_mutex.unlock();
            return;
        }
        task_state_channel = task_state_ch;
        auto [su, su_err] = client->telem.openWriter(WriterConfig{.channels = {task_state_ch.key}});
        if (err) {
            LOG(ERROR) << "Failed to open writer to update task state" << su_err.message();
            state_mutex.unlock();
            return;
        }
        state_updater = std::make_unique<Writer>(std::move(su));
    }
    auto fr = Frame(1);
    fr.add(task_state_channel.key, Series(std::vector{to_string(state.toJSON())}, JSON));
    if (!state_updater->write(std::move(fr))) {
        auto err = state_updater->close();
        LOG(ERROR) << "Failed to write task state update" << err.message();
        state_updater = nullptr;
    }
    state_mutex.unlock();
}
