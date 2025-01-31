/*
 * Copyright 2024 Redpanda Data, Inc.
 *
 * Licensed as a Redpanda Enterprise file under the Redpanda Community
 * License (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 * https://github.com/redpanda-data/redpanda/blob/master/licenses/rcl.md
 */
#include "datalake/record_translator.h"

#include "base/vlog.h"
#include "datalake/conversion_outcome.h"
#include "datalake/logger.h"
#include "datalake/table_definition.h"
#include "datalake/values_protobuf.h"
#include "iceberg/avro_utils.h"
#include "iceberg/datatypes.h"
#include "iceberg/values.h"
#include "iceberg/values_avro.h"

#include <avro/Generic.hh>
#include <avro/GenericDatum.hh>

namespace datalake {

namespace {

struct value_translating_visitor {
    // Buffer ready to be parsed, e.g. no schema ID or protobuf offsets.
    iobuf parsable_buf;
    const iceberg::field_type& type;

    ss::future<optional_value_outcome>
    operator()(const google::protobuf::Descriptor& d) {
        return deserialize_protobuf(std::move(parsable_buf), d);
    }
    ss::future<optional_value_outcome> operator()(const avro::ValidSchema& s) {
        avro::GenericDatum d(s);
        try {
            auto in = std::make_unique<iceberg::avro_iobuf_istream>(
              std::move(parsable_buf));
            auto decoder = avro::validatingDecoder(s, avro::binaryDecoder());
            decoder->init(*in);
            avro::decode(*decoder, d);
        } catch (...) {
            co_return value_conversion_exception(fmt::format(
              "Error reading Avro buffer: {}", std::current_exception()));
        }
        co_return iceberg::val_from_avro(d, type, iceberg::field_required::yes);
    }
};

} // namespace

std::ostream& operator<<(std::ostream& o, const record_translator::errc& e) {
    switch (e) {
    case record_translator::errc::translation_error:
        return o << "record_translator::errc::translation_error";
    }
}

record_type
record_translator::build_type(std::optional<resolved_type> val_type) {
    auto ret_type = schemaless_struct_type();
    std::optional<schema_identifier> val_id;
    if (val_type.has_value()) {
        // Add the extra user-defined fields.
        ret_type.fields.emplace_back(iceberg::nested_field::create(
          0,
          val_type->type_name,
          iceberg::field_required::no,
          std::move(val_type->type)));
        val_id = std::move(val_type->id);
    }
    return record_type{
      .comps = record_schema_components{
          .key_identifier = std::nullopt,
          .val_identifier = std::move(val_id),
      },
      .type = std::move(ret_type),
    };
}

ss::future<checked<iceberg::struct_value, record_translator::errc>>
record_translator::translate_data(
  kafka::offset o,
  iobuf key,
  const std::optional<resolved_type>& val_type,
  iobuf parsable_val,
  model::timestamp ts) {
    auto ret_data = iceberg::struct_value{};
    ret_data.fields.emplace_back(iceberg::long_value(o));
    // NOTE: Kafka uses milliseconds, Iceberg uses microseconds.
    ret_data.fields.emplace_back(iceberg::timestamp_value(ts.value() * 1000));
    ret_data.fields.emplace_back(iceberg::binary_value{std::move(key)});
    if (val_type.has_value()) {
        // Fill in the internal value field.
        ret_data.fields.emplace_back(std::nullopt);

        auto translated_val = co_await std::visit(
          value_translating_visitor{std::move(parsable_val), val_type->type},
          val_type->schema);
        if (translated_val.has_error()) {
            vlog(
              datalake_log.error,
              "Error converting buffer: {}",
              translated_val.error());
            // TODO: metric for data translation errors.
            // Either needs to drop the data or send it to a dead-letter queue.
            co_return errc::translation_error;
        }
        ret_data.fields.emplace_back(std::move(translated_val.value()));
    } else {
        ret_data.fields.emplace_back(
          iceberg::binary_value{std::move(parsable_val)});
    }
    co_return ret_data;
}

} // namespace datalake
