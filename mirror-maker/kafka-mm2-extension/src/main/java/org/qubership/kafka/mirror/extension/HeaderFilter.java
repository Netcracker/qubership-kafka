/*
 * Copyright 2024-2025 NetCracker Technology Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.qubership.kafka.mirror.extension;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.function.Predicate;
import java.util.regex.Pattern;
import javax.annotation.Nonnull;
import javax.annotation.Nullable;
import org.apache.commons.lang3.StringUtils;
import org.apache.kafka.common.config.ConfigDef;
import org.apache.kafka.common.config.ConfigDef.Importance;
import org.apache.kafka.common.config.ConfigDef.Type;
import org.apache.kafka.common.config.ConfigException;
import org.apache.kafka.connect.connector.ConnectRecord;
import org.apache.kafka.connect.header.Header;
import org.apache.kafka.connect.header.Headers;
import org.apache.kafka.connect.transforms.Transformation;
import org.apache.kafka.connect.transforms.util.SimpleConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * This class is designed to filter Kafka Connect records by headers.
 */
public class HeaderFilter<R extends ConnectRecord<R>> implements Transformation<R> {

  // visible for testing
  static final String TYPE_CONFIG = "filter.type";
  // visible for testing
  static final String HEADERS_CONFIG = "headers";
  // visible for testing
  static final String TOPICS_CONFIG = "topics";

  private static final Logger LOGGER = LoggerFactory.getLogger(HeaderFilter.class);

  private static final String TYPE_DOC =
      "The type of filter that can have \"include\" or \"exclude\" value.";
  private static final String HEADERS_DOC =
      "The comma-separated list of header in format: key or key=value.";
  private static final String TOPICS_DOC =
      "The comma-separated list of topics and/or regexes to filter records by headers.";

  private static final String ERROR_FORMAT = "Required configuration \"%s\" has empty value";

  private static final ConfigDef CONFIG_DEF = baseConfigDef();

  private Predicate<Headers> skipPredicate;
  private Pattern topicsPattern;

  private static ConfigDef baseConfigDef() {
    ConfigDef configDef = new ConfigDef();
    addConnectorConfigs(configDef);
    return configDef;
  }

  private static void addConnectorConfigs(ConfigDef configDef) {
    configDef.define(
            TYPE_CONFIG,
            Type.STRING,
            ConfigDef.NO_DEFAULT_VALUE,
            (name, value) -> {
              if (StringUtils.isBlank((CharSequence) value)) {
                throw new ConfigException(String.format(ERROR_FORMAT, name));
              }
            },
            Importance.HIGH,
            TYPE_DOC)
        .define(
            HEADERS_CONFIG,
            Type.LIST,
            ConfigDef.NO_DEFAULT_VALUE,
            (name, value) -> {
              if (value == null || ((List<?>) value).isEmpty()) {
                throw new ConfigException(String.format(ERROR_FORMAT, name));
              }
            },
            Importance.HIGH,
            HEADERS_DOC)
        .define(
            TOPICS_CONFIG,
            Type.LIST,
            "",
            Importance.MEDIUM,
            TOPICS_DOC);
  }

  // visible for testing
  @Nonnull
  static Map<String, Optional<String>> parseHeadersToFilter(@Nonnull List<String> headers) {
    Map<String, Optional<String>> result = new HashMap<>();
    for (String header : headers) {
      String[] keyValue = header.split("=");
      if (keyValue.length == 1) {
        result.put(keyValue[0], Optional.empty());
      } else if (keyValue.length == 2) {
        String value = keyValue[1];
        result.put(keyValue[0], Optional.ofNullable(nullIfEmpty(value)));
      } else {
        throw new ConfigException("Header '" + header + "' has invalid format.");
      }
    }
    return result;
  }

  @Nonnull
  private static Predicate<Headers> skipPredicate(
      @Nonnull FilterType filterType, @Nonnull Map<String, Optional<String>> headersToFilter) {
    switch (filterType) {
      case INCLUDE:
        return predicate(headersToFilter, false);
      case EXCLUDE:
        return predicate(headersToFilter, true);
      default:
        throw new IllegalArgumentException("Unknown filter type: " + filterType);
    }
  }

  @Nonnull
  private static Predicate<Headers> predicate(
      @Nonnull Map<String, Optional<String>> headersToFilter,
      boolean resultingValueWhenAtLeastOneHeaderMatches) {
    return headers -> {
      for (Header header : headers) {
        String headerKey = header.key();
        Optional<String> value = headersToFilter.get(headerKey);
        if (value != null) {
          Object headerValue = convert(header.value());
          LOGGER.debug("Record header '{}={}' needs to be checked by the filter",
              headerKey, headerValue);
          if (Objects.equals(headerValue, value.orElse(null))) {
            return resultingValueWhenAtLeastOneHeaderMatches;
          }
        }
      }
      return !resultingValueWhenAtLeastOneHeaderMatches;
    };
  }

  // visible for testing
  @Nullable
  static Object convert(@Nullable Object value) {
    if (value == null) {
      return null;
    }
    if (value instanceof byte[]) {
      return convertToString((byte[]) value);
    }
    if (value instanceof ByteBuffer) {
      return convertToString(((ByteBuffer) value).array());
    }
    if (value instanceof String) {
      return nullIfEmpty((String) value);
    }
    if (value instanceof Boolean) {
      return Boolean.toString((Boolean) value);
    }
    if (value instanceof Byte) {
      return Byte.toString((Byte) value);
    }
    if (value instanceof Short) {
      return Short.toString((Short) value);
    }
    if (value instanceof Integer) {
      return Integer.toString((Integer) value);
    }
    if (value instanceof Long) {
      return Long.toString((Long) value);
    }
    if (value instanceof Float) {
      return Float.toString((Float) value);
    }
    if (value instanceof Double) {
      return Double.toString((Double) value);
    }
    return value;
  }

  @Nullable
  private static String convertToString(byte[] value) {
    return nullIfEmpty(new String(value, StandardCharsets.UTF_8));
  }

  @Nullable
  private static String nullIfEmpty(@Nullable String value) {
    return StringUtils.defaultIfBlank(value, null);
  }

  @Nullable
  private static Pattern compileTopicsPattern(@Nonnull List<String> topics) {
    return topics.isEmpty() ? null : Pattern.compile(String.join("|", topics));
  }

  @Override
  public void configure(Map<String, ?> configs) {
    LOGGER.info("Configuring the header filter with settings: {}", configs);
    SimpleConfig config = new SimpleConfig(CONFIG_DEF, configs);
    FilterType filterType = FilterType.parse(config.getString(TYPE_CONFIG));
    Map<String, Optional<String>> headersToFilter =
        parseHeadersToFilter(config.getList(HEADERS_CONFIG));
    skipPredicate = skipPredicate(filterType, headersToFilter);
    topicsPattern = compileTopicsPattern(config.getList(TOPICS_CONFIG));
  }

  @Override
  public R apply(R record) {
    LOGGER.trace("Applying the header filter to record: {}", record);
    if (topicsPattern == null || topicsPattern.matcher(record.topic()).matches()) {
      Headers headers = record.headers();
      LOGGER.trace("Applying the filter to headers: {}", headers);
      if (skipPredicate.test(headers)) {
        LOGGER.debug("Record {} is skipped", record);
        return null;
      }
    }
    return record;
  }

  @Override
  public ConfigDef config() {
    return CONFIG_DEF;
  }

  @Override
  public void close() {
    LOGGER.trace("Closing the header filter");
  }

  private enum FilterType {

    INCLUDE("include"),

    EXCLUDE("exclude");

    @Nonnull
    private final String value;

    FilterType(@Nonnull String value) {
      this.value = value;
    }

    @Nonnull
    private static FilterType parse(@Nonnull String value) {
      for (FilterType type : FilterType.values()) {
        if (type.value.equals(value)) {
          return type;
        }
      }
      throw new IllegalArgumentException("Unknown filter type: " + value);
    }
  }
}
