// Copyright 2024 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

#include "config/config_store.h"
#include "config/property.h"
#include "config/types.h"

#include <gtest/gtest.h>
#include <yaml-cpp/yaml.h>

#include <iostream>

namespace config {
namespace {

struct test_config : public config_store {
    enterprise<property<bool>> enterprise_bool;
    enterprise<enum_property<ss::sstring>> enterprise_str_enum;
    enterprise<property<std::vector<ss::sstring>>> enterprise_str_vec;
    enterprise<property<std::optional<int>>> enterprise_opt_int;
    enterprise<enum_property<tls_version>> enterprise_enum;

    using meta = base_property::metadata;

    test_config()
      : enterprise_bool(
          *this,
          true,
          "enterprise_bool",
          "An enterprise-only bool config",
          meta{},
          false,
          property<bool>::noop_validator,
          std::nullopt)
      , enterprise_str_enum(
          *this,
          std::vector<ss::sstring>{"bar"},
          "enterprise_str_enum",
          "An enterprise-only enum property",
          meta{},
          "foo",
          std::vector<ss::sstring>{"foo", "bar", "baz"})
      , enterprise_str_vec(
          *this,
          std::vector<ss::sstring>{"GSSAPI"},
          "enterprise_str_vec",
          "An enterprise-only vector of strings")
      , enterprise_opt_int(
          *this,
          [](const int& v) -> bool { return v > 1000; },
          "enterprise_opt_int",
          "An enterprise-only optional int",
          meta{},
          0)
      , enterprise_enum(
          *this,
          std::vector<tls_version>{tls_version::v1_3},
          "enterprise_str_enum",
          "An enterprise-only enum property",
          meta{},
          tls_version::v1_1,
          std::vector<tls_version>{
            tls_version::v1_0,
            tls_version::v1_1,
            tls_version::v1_2,
            tls_version::v1_3}) {}
};

} // namespace

TEST(EnterprisePropertyTest, TestRestriction) {
    using N = YAML::Node;
    test_config cfg;

    EXPECT_FALSE(cfg.enterprise_bool.check_restricted(N(false)));
    EXPECT_TRUE(cfg.enterprise_bool.check_restricted(N(true)));

    EXPECT_FALSE(cfg.enterprise_str_enum.check_restricted(N("foo")));
    EXPECT_TRUE(cfg.enterprise_str_enum.check_restricted(N("bar")));

    EXPECT_FALSE(cfg.enterprise_str_vec.check_restricted(
      N(std::vector<ss::sstring>{"foo", "bar", "baz"})));
    EXPECT_TRUE(cfg.enterprise_str_vec.check_restricted(
      N(std::vector<ss::sstring>{"foo", "bar", "baz", "GSSAPI"})));

    EXPECT_FALSE(cfg.enterprise_opt_int.check_restricted(N(10)));
    EXPECT_TRUE(cfg.enterprise_opt_int.check_restricted(N(10000)));

    EXPECT_FALSE(cfg.enterprise_enum.check_restricted(N(tls_version::v1_0)));
    EXPECT_TRUE(cfg.enterprise_enum.check_restricted(N(tls_version::v1_3)));
}

TEST(EnterprisePropertyTest, TestTypeName) {
    test_config cfg;
    EXPECT_EQ(cfg.enterprise_bool.type_name(), "boolean");
    EXPECT_EQ(cfg.enterprise_str_enum.type_name(), "string");
    EXPECT_EQ(cfg.enterprise_str_vec.type_name(), "string");
    EXPECT_EQ(cfg.enterprise_opt_int.type_name(), "integer");
    EXPECT_EQ(cfg.enterprise_enum.type_name(), "string");
}

} // namespace config
