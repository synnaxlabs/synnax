#pragma once

#include <numbers>
#include <string>

#include "v1/framer.pb.h"

namespace Synnax {
    namespace Telem {
        typedef uint8_t Authority;

        struct Subject {
            std::string name;
            std::string key;

            void to_proto(api::v1::ControlSubject *s) const {
                s->set_name(name);
                s->set_key(key);
            }
        };
    }
}



