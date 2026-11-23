// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#ifndef _WIN32
#include <pwd.h>
#include <unistd.h>
#endif

#include "gtest/gtest.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/env/env.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief it should keep the state of a user in the folder of that user while the
/// Driver service is not installed.
TEST(StateTest, UserStateWithoutService) {
#ifdef __linux__
    GTEST_SKIP() << "Linux keeps the state of every process in the folder of the "
                    "computer";
#endif
    const auto home = std::filesystem::temp_directory_path() / "rack-test-home";
    std::filesystem::remove_all(home);
#ifdef _WIN32
    const std::string var = "LOCALAPPDATA";
    const auto expected = home / "synnax-driver";
#else
    const std::string var = "HOME";
    const auto expected = home / "Library" / "Application Support" / "synnax-driver";
#endif
    const char *previous = std::getenv(var.c_str());
    const char *sudo_user = std::getenv("SUDO_USER");
    x::defer::defer restore([&,
                             previous = std::string(previous ? previous : ""),
                             sudo_user = std::string(sudo_user ? sudo_user : "")] {
        x::env::set(var, previous);
        if (!sudo_user.empty()) x::env::set("SUDO_USER", sudo_user);
        std::filesystem::remove_all(home);
    });
    x::env::set(var, home.string());
    x::env::unset("SUDO_USER");
    auto args = x::args::Parser(std::vector<std::string>{"program"});
    Config cfg;
    ASSERT_NIL(cfg.load_persisted_state(args));
    ASSERT_EQ(cfg.state_dir, expected);
}

/// @brief it should refuse the state of a user to root under sudo, before it changes
/// anything, and use it for any other process.
TEST(StateTest, RefuseUserStateUnderSudo) {
#if defined(__linux__) || defined(_WIN32)
    GTEST_SKIP() << "only macOS keeps the state of a user in the folder of that user";
#else
    const auto home = std::filesystem::temp_directory_path() / "rack-test-sudo-home";
    std::filesystem::remove_all(home);
    const char *previous = std::getenv("HOME");
    const char *sudo_user = std::getenv("SUDO_USER");
    x::defer::defer restore([&,
                             previous = std::string(previous ? previous : ""),
                             sudo_user = std::string(sudo_user ? sudo_user : "")] {
        x::env::set("HOME", previous);
        if (sudo_user.empty())
            x::env::unset("SUDO_USER");
        else
            x::env::set("SUDO_USER", sudo_user);
        std::filesystem::remove_all(home);
    });
    x::env::set("HOME", home.string());
    // A user that does not exist keeps the state directory in HOME.
    x::env::set("SUDO_USER", "rack-test-no-such-user");
    auto args = x::args::Parser(std::vector<std::string>{"program"});
    Config cfg;
    if (geteuid() != 0) {
        ASSERT_NIL(Config::secure_persisted_state(args));
        ASSERT_NIL(cfg.load_persisted_state(args));
        return;
    }
    ASSERT_OCCURRED_AS(Config::secure_persisted_state(args), STATE_ERROR);
    const auto err = cfg.load_persisted_state(args);
    ASSERT_OCCURRED_AS(err, STATE_ERROR);
    ASSERT_NE(err.message().find("without sudo"), std::string::npos);
    ASSERT_FALSE(std::filesystem::exists(home));
#endif
}

/// @brief it should let root keep its own state when root runs sudo.
TEST(StateTest, RootStateUnderSudoAsRoot) {
#if defined(__linux__) || defined(_WIN32)
    GTEST_SKIP() << "only macOS keeps the state of a user in the folder of that user";
#else
    const passwd *root = getpwuid(0);
    if (geteuid() != 0 || root == nullptr) GTEST_SKIP() << "only root owns its state";
    const auto dir = std::filesystem::path(root->pw_dir) / "Library" /
                     "Application Support" / "synnax-driver";
    if (std::filesystem::exists(dir)) GTEST_SKIP() << dir << " holds a saved login";
    const std::string name = root->pw_name;
    const char *sudo_user = std::getenv("SUDO_USER");
    x::defer::defer restore([&, sudo_user = std::string(sudo_user ? sudo_user : "")] {
        if (sudo_user.empty())
            x::env::unset("SUDO_USER");
        else
            x::env::set("SUDO_USER", sudo_user);
        std::filesystem::remove_all(dir);
    });
    x::env::set("SUDO_USER", name);
    auto args = x::args::Parser(std::vector<std::string>{"program"});
    Config cfg;
    ASSERT_NIL(cfg.load_persisted_state(args));
    ASSERT_EQ(cfg.state_dir, dir);
#endif
}

namespace fs = std::filesystem;

/// @brief runs each spec with the default state path, and with --state-file.
class StateModeTest : public ::testing::TestWithParam<bool> {
protected:
    fs::path home = fs::temp_directory_path() / "rack-state-mode-test";
    fs::path path;
    x::args::Parser args;
    std::string previous_home;
    std::string sudo_user;

    void SetUp() override {
#ifdef _WIN32
        GTEST_SKIP() << "Windows files have no POSIX mode";
#endif
        fs::remove_all(this->home);
        const char *h = std::getenv("HOME");
        const char *s = std::getenv("SUDO_USER");
        this->previous_home = h != nullptr ? h : "";
        this->sudo_user = s != nullptr ? s : "";
        if (!GetParam()) {
            this->path = this->home / "state" / "persisted-state.json";
            this->args = x::args::Parser(
                std::vector<std::string>{"program", "--state-file", this->path.string()}
            );
            return;
        }
#ifdef __linux__
        GTEST_SKIP() << "the default path on Linux is the directory of the computer";
#endif
        x::env::set("HOME", this->home.string());
        x::env::unset("SUDO_USER");
        this->path = this->home / "Library" / "Application Support" / "synnax-driver" /
                     "persisted-state.json";
        this->args = x::args::Parser(std::vector<std::string>{"program"});
    }

    void TearDown() override {
        if (GetParam()) {
            x::env::set("HOME", this->previous_home);
            if (!this->sudo_user.empty()) x::env::set("SUDO_USER", this->sudo_user);
        }
        fs::remove_all(this->home);
    }

    [[nodiscard]] fs::perms mode(const fs::path &p) const {
        return fs::status(p).permissions() & fs::perms::all;
    }
};

INSTANTIATE_TEST_SUITE_P(Paths, StateModeTest, ::testing::Values(false, true));

/// @brief it should create a missing state directory and file for the owner alone.
TEST_P(StateModeTest, CreateMissingOwnerOnly) {
    Config cfg;
    ASSERT_NIL(cfg.load_persisted_state(this->args));
    ASSERT_EQ(this->mode(this->path.parent_path()), fs::perms::owner_all);
    ASSERT_EQ(this->mode(this->path), fs::perms::owner_read | fs::perms::owner_write);
}

/// @brief it should keep the mode of an existing state directory.
TEST_P(StateModeTest, KeepDirMode) {
    const auto dir = this->path.parent_path();
    fs::create_directories(dir);
    const auto dir_mode = fs::perms::owner_all | fs::perms::group_read |
                          fs::perms::group_exec;
    fs::permissions(dir, dir_mode);
    Config cfg;
    ASSERT_NIL(cfg.load_persisted_state(this->args));
    ASSERT_EQ(this->mode(dir), dir_mode);
}

/// @brief it should refuse a state file that other users can read, and keep its mode.
TEST_P(StateModeTest, RefuseReadableStateFile) {
    fs::create_directories(this->path.parent_path());
    std::ofstream(this->path) << "{}";
    const auto file_mode = fs::perms::owner_read | fs::perms::owner_write |
                           fs::perms::group_read;
    fs::permissions(this->path, file_mode);
    Config cfg;
    const auto err = cfg.load_persisted_state(this->args);
    ASSERT_OCCURRED_AS(err, STATE_ERROR);
    ASSERT_NE(
        err.message().find("chmod 600 " + this->path.string()),
        std::string::npos
    );
    ASSERT_NE(err.message().find("fsGroupChangePolicy"), std::string::npos);
    ASSERT_EQ(this->mode(this->path), file_mode);
}

/// @brief it should change no mode when a process without root secures the state.
TEST_P(StateModeTest, SecureWithoutRootKeepsModes) {
#ifndef _WIN32
    if (geteuid() == 0) GTEST_SKIP() << "root changes modes";
#endif
    const auto dir = this->path.parent_path();
    fs::create_directories(dir);
    const auto dir_mode = fs::perms::owner_all | fs::perms::group_read |
                          fs::perms::group_exec;
    fs::permissions(dir, dir_mode);
    std::ofstream(this->path) << "{}";
    const auto file_mode = fs::perms::owner_read | fs::perms::owner_write |
                           fs::perms::group_read;
    fs::permissions(this->path, file_mode);
    ASSERT_NIL(Config::secure_persisted_state(this->args));
    ASSERT_EQ(this->mode(dir), dir_mode);
    ASSERT_EQ(this->mode(this->path), file_mode);
}

/// @brief it should change only the state file, never its directory, when root
/// secures a --state-file.
TEST_P(StateModeTest, SecureStateFileKeepsDir) {
#ifndef _WIN32
    if (geteuid() != 0) GTEST_SKIP() << "only root changes modes";
#endif
    if (GetParam()) GTEST_SKIP() << "the default directory is dedicated";
    const auto dir = this->path.parent_path();
    fs::create_directories(dir);
    const auto dir_mode = fs::perms::owner_all | fs::perms::group_read |
                          fs::perms::group_exec | fs::perms::others_exec;
    fs::permissions(dir, dir_mode);
    std::ofstream(this->path) << "{}";
    ASSERT_NIL(Config::secure_persisted_state(this->args));
    ASSERT_EQ(this->mode(dir), dir_mode);
    ASSERT_FALSE(fs::exists(dir / OPCUA_STATE_DIR));
}

/// @brief it should refuse a state file that another user owns.
TEST_P(StateModeTest, RefuseStateFileOfAnotherUser) {
#ifndef _WIN32
    if (geteuid() != 0) GTEST_SKIP() << "only root can give a file to another user";
    const passwd *nobody = getpwnam("nobody");
    if (nobody == nullptr) GTEST_SKIP() << "no nobody user";
    fs::create_directories(this->path.parent_path());
    std::ofstream(this->path) << "{}";
    fs::permissions(this->path, fs::perms::owner_read | fs::perms::owner_write);
    ASSERT_EQ(chown(this->path.c_str(), nobody->pw_uid, nobody->pw_gid), 0);
    Config cfg;
    const auto err = cfg.load_persisted_state(this->args);
    ASSERT_OCCURRED_AS(err, STATE_ERROR);
    ASSERT_NE(err.message().find("Run the Driver as nobody"), std::string::npos);
    ASSERT_EQ(err.message().find("chown"), std::string::npos);
#endif
}

/// @brief it should refuse a readable state file given by a bare file name.
TEST(StateTest, RefuseReadableRelativeStateFile) {
#ifdef _WIN32
    GTEST_SKIP() << "Windows files have no POSIX mode";
#endif
    const auto dir = fs::temp_directory_path() / "rack-state-relative-test";
    fs::remove_all(dir);
    fs::create_directories(dir);
    const auto previous = fs::current_path();
    x::defer::defer restore([&] {
        fs::current_path(previous);
        fs::remove_all(dir);
    });
    fs::current_path(dir);
    std::ofstream("state.json") << "{}";
    fs::permissions(
        "state.json",
        fs::perms::owner_read | fs::perms::owner_write | fs::perms::group_read
    );
    auto args = x::args::Parser(
        std::vector<std::string>{"program", "--state-file", "state.json"}
    );
    Config cfg;
    ASSERT_OCCURRED_AS(cfg.load_persisted_state(args), STATE_ERROR);
}
}
