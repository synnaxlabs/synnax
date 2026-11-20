// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/fs/fs.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/testutil/testutil.h"

#include "driver/file/file.h"

namespace driver::file {
namespace {
void write(const std::filesystem::path &path, const std::string &data) {
    std::ofstream out(path, std::ios::binary);
    out << data;
}

/// @brief the details of x::x509::testutil::CERTIFICATE, as the Core stores them.
const x::json::json CERTIFICATE_DETAILS = {
    {"subject", "CN=Test Client,O=Synnax"},
    {"issuer", "CN=Test Client,O=Synnax"},
    {"not_before", "1767225600000000000"},
    {"not_after", "4922899199000000000"},
    {"sha1", "915d90fff8fb63289202f5155ce6cbc027cc1c57"},
    {"sha256", "7b42752d0bb377f9c35ac660d491fa8fa46e5b8885fd41a2650f19f8bd52309e"},
};

/// @brief a certificate that mbedTLS refuses, because its subject holds a
/// NumericString.
const std::string NUMERIC_CERTIFICATE = R"(-----BEGIN CERTIFICATE-----
MIIBSjCB8aADAgECAgEIMAoGCCqGSM49BAMCMC8xEDAOBgNVBAMTB251bWVyaWMx
DTALBgNVBAUSBDQyIDcxDDAKBgMqAwgSAzEyMzAeFw0yNjAxMDEwMDAwMDBaFw0y
NzAxMDEwMDAwMDBaMC8xEDAOBgNVBAMTB251bWVyaWMxDTALBgNVBAUSBDQyIDcx
DDAKBgMqAwgSAzEyMzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABLw4lkTtJ/T4
PO71liCaQg46+tafvX0aW2cviMj+52AwyzDyCnTygLgGh5Q76IMApJk/WSpERDYA
lsmhwz1QY9kwCgYIKoZIzj0EAwIDSAAwRQIgCEN+/A/qTvEOW1/LsgB3i1rmb2BJ
hhkSnCbERKf/xM8CIQDFBenaYI7T9bTTfwDAo6a/4katmokiJn5czkoG/d7M6Q==
-----END CERTIFICATE-----
)";

/// @returns the entry named name in a reply, or null.
x::json::json entry(const x::json::json &data, const std::string &name) {
    for (const auto &e: data["entries"])
        if (e["name"] == name) return e;
    return nullptr;
}

class BrowserTest : public ::testing::Test {
protected:
    std::filesystem::path dir;
    std::filesystem::path secrets;

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("file-browser-" + x::uuid::create().to_string());
        this->secrets = this->dir / "secrets";
        std::filesystem::create_directories(this->secrets);
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    [[nodiscard]] Browser browser() const {
        return Browser(
            Config{
                .roots = {this->dir.root_path()},
                .home = this->dir / "home",
                .secrets_dir = this->secrets,
            }
        );
    }
};
}

/// @brief it should list the roots, the home folder, and the secrets folder at the
/// top level.
TEST_F(BrowserTest, TopLevel) {
    const auto data = ASSERT_NIL_P(this->browser().list(""));
    const x::json::json expected = {
        {"path", ""},
        {"parent", ""},
        {"secrets_dir", this->secrets.string()},
        {"secret", false},
        {"truncated", false},
        {"entries",
         {
             {{"name", this->dir.root_path().string()},
              {"path", this->dir.root_path().string()},
              {"kind", "directory"}},
             {{"name", "home"},
              {"path", (this->dir / "home").string()},
              {"kind", "directory"}},
             {{"name", "secrets"},
              {"path", this->secrets.string()},
              {"kind", "directory"}},
         }},
    };
    ASSERT_EQ(data, expected);
}

/// @brief it should leave the secrets folder out of the top level when none is set.
TEST_F(BrowserTest, TopLevelWithoutSecrets) {
    const Browser browser(Config{.roots = {"/"}});
    const auto data = ASSERT_NIL_P(browser.list(""));
    ASSERT_EQ(data["secrets_dir"], "");
    ASSERT_EQ(data["entries"].size(), 1);
    ASSERT_EQ(data["entries"][0]["path"], "/");
}

/// @brief it should list folders first, then files, each in byte order of name.
TEST_F(BrowserTest, ListOrder) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder / "a");
    std::filesystem::create_directories(folder / "B");
    write(folder / "a.txt", "hello");
    write(folder / "Z.txt", "");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    std::vector<std::string> names;
    for (const auto &e: data["entries"])
        names.push_back(e["name"]);
    ASSERT_EQ(names, std::vector<std::string>({"B", "a", "Z.txt", "a.txt"}));
    ASSERT_EQ(data["path"], folder.string());
    ASSERT_EQ(data["parent"], this->dir.string());
    ASSERT_EQ(data["secrets_dir"], this->secrets.string());
    ASSERT_EQ(data["secret"], false);
    ASSERT_EQ(data["truncated"], false);
}

/// @brief it should give each entry its kind, size, and modified time.
TEST_F(BrowserTest, EntryFields) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder / "sub");
    write(folder / "a.txt", "hello");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    const auto file = entry(data, "a.txt");
    ASSERT_EQ(file["path"], (folder / "a.txt").string());
    ASSERT_EQ(file["kind"], "file");
    ASSERT_EQ(file["size"], 5);
    const auto modified = to_timestamp(
        std::filesystem::last_write_time(folder / "a.txt")
    );
    ASSERT_EQ(file["modified"], std::to_string(modified.nanoseconds()));
    ASSERT_FALSE(file.contains("certificate"));
    const auto sub = entry(data, "sub");
    ASSERT_EQ(sub["kind"], "directory");
    ASSERT_EQ(sub["size"], 0);
    ASSERT_TRUE(sub.contains("modified"));
}

/// @brief it should give the details of a certificate file, as the Core stores them.
TEST_F(BrowserTest, CertificateDetails) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    write(folder / "server.pem", x::x509::testutil::CERTIFICATE);
    write(folder / "upper.CRT", x::x509::testutil::CERTIFICATE);
    write(
        folder / "bundle.pem",
        "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n" +
            x::x509::testutil::CERTIFICATE
    );
    write(folder / "broken.cer", "not a certificate");
    write(folder / "numeric.pem", NUMERIC_CERTIFICATE);
    write(folder / "server.txt", x::x509::testutil::CERTIFICATE);
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    ASSERT_EQ(entry(data, "server.pem")["certificate"], CERTIFICATE_DETAILS);
    ASSERT_EQ(entry(data, "upper.CRT")["certificate"], CERTIFICATE_DETAILS);
    ASSERT_EQ(entry(data, "bundle.pem")["certificate"], CERTIFICATE_DETAILS);
    ASSERT_FALSE(entry(data, "broken.cer").contains("certificate"));
    const auto numeric = entry(data, "numeric.pem");
    ASSERT_EQ(numeric["size"], NUMERIC_CERTIFICATE.size());
    ASSERT_FALSE(numeric.contains("certificate"));
    ASSERT_FALSE(entry(data, "server.txt").contains("certificate"));
}

/// @brief it should list folders in full when the secrets folder does not exist.
TEST_F(BrowserTest, MissingSecretsFolder) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    write(folder / "server.pem", x::x509::testutil::CERTIFICATE);
    const Browser browser(Config{.secrets_dir = this->dir / "absent"});
    const auto data = ASSERT_NIL_P(browser.list(folder.string()));
    ASSERT_FALSE(data["secret"].get<bool>());
    ASSERT_EQ(entry(data, "server.pem")["certificate"], CERTIFICATE_DETAILS);
}

/// @brief it should not read a certificate file larger than 1 MiB.
TEST_F(BrowserTest, LargeCertificateNotRead) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    write(
        folder / "big.pem",
        x::x509::testutil::CERTIFICATE + std::string(1 << 20, '\n')
    );
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    const auto big = entry(data, "big.pem");
    ASSERT_GT(big["size"].get<std::uintmax_t>(), MAX_CERTIFICATE_SIZE);
    ASSERT_FALSE(big.contains("certificate"));
}

/// @brief it should give only the name, path, and kind of each entry in the secrets
/// folder.
TEST_F(BrowserTest, SecretsFolderNamesOnly) {
    std::filesystem::create_directories(this->secrets / "nested");
    write(this->secrets / "password", "hunter2");
    write(this->secrets / "client.pem", x::x509::testutil::CERTIFICATE);
    const auto data = ASSERT_NIL_P(this->browser().list(this->secrets.string()));
    ASSERT_EQ(data["secret"], true);
    const x::json::json expected = {
        {{"name", "nested"},
         {"path", (this->secrets / "nested").string()},
         {"kind", "directory"}},
        {{"name", "client.pem"},
         {"path", (this->secrets / "client.pem").string()},
         {"kind", "file"}},
        {{"name", "password"},
         {"path", (this->secrets / "password").string()},
         {"kind", "file"}},
    };
    ASSERT_EQ(data["entries"], expected);
    const auto nested = ASSERT_NIL_P(
        this->browser().list((this->secrets / "nested").string())
    );
    ASSERT_EQ(nested["secret"], true);
}

/// @brief it should match the secrets folder by file identity, so another case of its
/// name matches on a file system that ignores case.
TEST_F(BrowserTest, SecretsFolderOtherCase) {
    write(this->secrets / "password", "hunter2");
    const auto upper = this->dir / "SECRETS";
    if (!std::filesystem::exists(upper))
        GTEST_SKIP() << "the file system is case-sensitive";
    const auto data = ASSERT_NIL_P(this->browser().list(upper.string()));
    ASSERT_EQ(data["secret"], true);
    ASSERT_EQ(entry(data, "password").size(), 3);
}

/// @brief it should treat a folder link to the secrets folder or to a folder in it as
/// a secrets folder, and give only the name of a link to a folder in it.
TEST_F(BrowserTest, FolderLinkToSecrets) {
    write(this->secrets / "password", "hunter2");
    std::filesystem::create_directories(this->secrets / "nested");
    write(this->secrets / "nested" / "client.pem", x::x509::testutil::CERTIFICATE);
    std::error_code ec;
    std::filesystem::create_directory_symlink(this->secrets, this->dir / "alias", ec);
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    std::filesystem::create_directory_symlink(
        this->secrets / "nested",
        this->dir / "inner",
        ec
    );
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto alias = ASSERT_NIL_P(
        this->browser().list((this->dir / "alias").string())
    );
    ASSERT_EQ(alias["secret"], true);
    ASSERT_EQ(entry(alias, "password").size(), 3);
    const auto data = ASSERT_NIL_P(this->browser().list(this->dir.string()));
    ASSERT_EQ(data["secret"], false);
    ASSERT_EQ(entry(data, "inner").size(), 3);
    const auto inner = ASSERT_NIL_P(
        this->browser().list((this->dir / "inner").string())
    );
    ASSERT_EQ(inner["secret"], true);
    ASSERT_EQ(entry(inner, "client.pem").size(), 3);
}

/// @brief it should give only the name of the secrets folder and of a link into it.
TEST_F(BrowserTest, LinkIntoSecretsNamesOnly) {
    write(this->secrets / "client.pem", x::x509::testutil::CERTIFICATE);
    std::error_code ec;
    std::filesystem::create_symlink(
        this->secrets / "client.pem",
        this->dir / "link.pem",
        ec
    );
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto data = ASSERT_NIL_P(this->browser().list(this->dir.string()));
    ASSERT_EQ(data["secret"], false);
    const auto link = entry(data, "link.pem");
    ASSERT_EQ(link.size(), 3);
    ASSERT_EQ(link["kind"], "file");
    ASSERT_EQ(entry(data, "secrets").size(), 3);
}

/// @brief it should read a link outside the secrets folder from its resolved target.
TEST_F(BrowserTest, LinkReadFromTarget) {
    const auto certs = this->dir / "certs";
    std::filesystem::create_directories(certs);
    write(certs / "server.pem", x::x509::testutil::CERTIFICATE);
    std::error_code ec;
    std::filesystem::create_symlink(certs / "server.pem", this->dir / "link.pem", ec);
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto data = ASSERT_NIL_P(this->browser().list(this->dir.string()));
    const auto link = entry(data, "link.pem");
    ASSERT_EQ(link["path"], (this->dir / "link.pem").string());
    ASSERT_EQ(link["size"], x::x509::testutil::CERTIFICATE.size());
    const auto modified = to_timestamp(
        std::filesystem::last_write_time(certs / "server.pem")
    );
    ASSERT_EQ(link["modified"], std::to_string(modified.nanoseconds()));
    ASSERT_EQ(link["certificate"], CERTIFICATE_DETAILS);
}

/// @brief it should leave out a link whose target is missing.
TEST_F(BrowserTest, BrokenLinkLeftOut) {
    std::error_code ec;
    std::filesystem::create_symlink(this->dir / "missing", this->dir / "broken", ec);
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto data = ASSERT_NIL_P(this->browser().list(this->dir.string()));
    ASSERT_TRUE(entry(data, "broken").is_null());
}

/// @brief it should resolve the links of the secrets folder at each list, so a link
/// made after the browser starts still hides the files.
TEST_F(BrowserTest, SecretsLinkResolvedEachList) {
    const auto link = this->dir / "link";
    const Browser browser(Config{.secrets_dir = link});
    const auto real = this->dir / "real";
    std::filesystem::create_directories(real);
    write(real / "client.pem", x::x509::testutil::CERTIFICATE);
    std::error_code ec;
    std::filesystem::create_directory_symlink(real, link, ec);
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto data = ASSERT_NIL_P(browser.list(real.string()));
    ASSERT_EQ(data["secret"], true);
    ASSERT_EQ(entry(data, "client.pem").size(), 3);
}

/// @brief it should leave out an entry whose name is not valid UTF-8.
TEST_F(BrowserTest, NonUTF8NameLeftOut) {
#ifndef __linux__
    GTEST_SKIP() << "only Linux keeps a file name that is not valid UTF-8";
#endif
    const auto folder = this->dir / "names";
    std::filesystem::create_directories(folder);
    write(folder / "ok.txt", "");
    write(folder / std::string("caf\xE9.txt"), "");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    ASSERT_EQ(data["entries"].size(), 1);
    ASSERT_EQ(data["entries"][0]["name"], "ok.txt");
}

/// @brief it should keep the first 1000 entries of a larger folder.
TEST_F(BrowserTest, Truncated) {
    const auto folder = this->dir / "many";
    std::filesystem::create_directories(folder);
    for (size_t i = 0; i <= MAX_ENTRIES; i++)
        write(folder / ("f" + std::to_string(10000 + i)), "");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    ASSERT_EQ(data["truncated"], true);
    ASSERT_EQ(data["entries"].size(), MAX_ENTRIES);
    ASSERT_EQ(data["entries"][0]["name"], "f10000");
}

/// @brief it should give the folder without a trailing separator or dot segments.
TEST_F(BrowserTest, NormalPath) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    const auto data = ASSERT_NIL_P(
        this->browser().list((this->dir / "secrets" / ".." / "certs").string() + "/")
    );
    ASSERT_EQ(data["path"], folder.string());
}

/// @brief it should give a root no parent.
TEST_F(BrowserTest, RootHasNoParent) {
    const auto root = this->dir.root_path().string();
    const auto data = ASSERT_NIL_P(this->browser().list(root));
    ASSERT_EQ(data["path"], root);
    ASSERT_EQ(data["parent"], "");
}

/// @brief it should name a home folder that is a root by its path.
TEST_F(BrowserTest, RootHomeNamedByPath) {
    const auto root = this->dir.root_path();
    const Browser browser(Config{.home = root});
    const auto data = ASSERT_NIL_P(browser.list(""));
    ASSERT_EQ(data["entries"][0]["name"], root.string());
}

/// @brief it should list a home folder that is one of the roots only once.
TEST_F(BrowserTest, RootHomeListedOnce) {
    const auto root = this->dir.root_path();
    const Browser browser(Config{.roots = {root}, .home = root});
    const auto data = ASSERT_NIL_P(browser.list(""));
    ASSERT_EQ(data["entries"].size(), 1);
    ASSERT_EQ(data["entries"][0]["path"], root.string());
}

/// @brief it should reject a folder that the Driver cannot read.
TEST_F(BrowserTest, PermissionDenied) {
    const auto locked = this->dir / "locked";
    std::filesystem::create_directories(locked);
    std::filesystem::permissions(locked, std::filesystem::perms::none);
    x::defer::defer restore([&] {
        std::filesystem::permissions(locked, std::filesystem::perms::owner_all);
    });
    std::error_code ec;
    std::filesystem::directory_iterator probe(locked, ec);
    if (!ec) GTEST_SKIP() << "cannot make a folder unreadable";
    const auto [data, err] = this->browser().list(locked.string());
    ASSERT_MATCHES(err, x::fs::PERMISSION_DENIED);
    ASSERT_EQ(err.data, "The Driver cannot read " + locked.string() + ".");
}

/// @brief it should reject a relative path.
TEST_F(BrowserTest, RelativePath) {
    const auto [data, err] = this->browser().list("certs");
    ASSERT_MATCHES(err, x::fs::INVALID_PATH);
    ASSERT_EQ(err.data, "certs is not an absolute path.");
}

/// @brief it should reject a path where nothing is.
TEST_F(BrowserTest, MissingPath) {
    const auto missing = this->dir / "missing";
    const auto [data, err] = this->browser().list(missing.string());
    ASSERT_MATCHES(err, x::fs::NOT_FOUND);
    ASSERT_EQ(err.data, missing.string() + " does not exist.");
}

/// @brief it should reject the path of a file.
TEST_F(BrowserTest, FilePath) {
    write(this->dir / "a.txt", "");
    const auto file = this->dir / "a.txt";
    const auto [data, err] = this->browser().list(file.string());
    ASSERT_MATCHES(err, x::fs::INVALID_PATH);
    ASSERT_EQ(err.data, file.string() + " is not a folder.");
}
}
