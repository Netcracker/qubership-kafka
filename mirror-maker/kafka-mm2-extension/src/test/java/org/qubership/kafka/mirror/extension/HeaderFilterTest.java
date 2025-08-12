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

import static org.qubership.kafka.mirror.extension.HeaderFilter.HEADERS_CONFIG;
import static org.qubership.kafka.mirror.extension.HeaderFilter.TOPICS_CONFIG;
import static org.qubership.kafka.mirror.extension.HeaderFilter.TYPE_CONFIG;
import static org.qubership.kafka.mirror.extension.HeaderFilter.convert;
import static org.qubership.kafka.mirror.extension.HeaderFilter.parseHeadersToFilter;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.equalTo;
import static org.hamcrest.Matchers.hasEntry;
import static org.hamcrest.Matchers.nullValue;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import javax.annotation.Nonnull;
import javax.annotation.Nullable;
import org.apache.kafka.common.config.ConfigException;
import org.apache.kafka.connect.header.ConnectHeaders;
import org.apache.kafka.connect.header.Headers;
import org.apache.kafka.connect.source.SourceRecord;
import org.junit.Test;

public class HeaderFilterTest {

  private static final String EXCLUDE = "exclude";
  private static final String INCLUDE = "include";
  private static final String MESSAGE_TYPE = "messageType";
  private static final String HEARTBEAT = "heartbeat";

  @Nonnull
  private static SourceRecord createSourceRecord(@Nullable Headers headers) {
    return new SourceRecord(null, null, "test-topic", 0, null, null, null, null, 0L, headers);
  }

  @Test(expected = ConfigException.class)
  public void testConfigureWhenFilterTypeConfigIsNotSpecified() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(HEADERS_CONFIG, HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
    }
  }

  @Test(expected = ConfigException.class)
  public void testConfigureWhenFilterTypeConfigHasNullValue() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, null);
      props.put(HEADERS_CONFIG, HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
    }
  }

  @Test(expected = ConfigException.class)
  public void testConfigureWhenFilterTypeConfigHasEmptyStringValue() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, "");
      props.put(HEADERS_CONFIG, HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
    }
  }

  @Test(expected = ConfigException.class)
  public void testConfigureWhenHeadersConfigIsNotSpecified() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
    }
  }

  @Test(expected = ConfigException.class)
  public void testConfigureWhenHeadersConfigHasNullValue() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, null);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
    }
  }

  @Test(expected = ConfigException.class)
  public void testConfigureWhenHeadersConfigHasEmptyStringValue() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, "");
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreNotSpecifiedAndHeadersAreMissingInRecord() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, HEARTBEAT);
      filter.configure(props);
      SourceRecord sourceRecord = createSourceRecord(null);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreNotSpecifiedAndHeaderKeyIsMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      filter.configure(props);
      Headers headers = new ConnectHeaders()
          .addString(MESSAGE_TYPE, HEARTBEAT + "V1")
          .add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreNotSpecifiedAndOnlyHeaderKeyIsMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreNotSpecifiedAndHeaderKeyValuePairIsMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      filter.configure(props);
      Headers headers = new ConnectHeaders().addString(MESSAGE_TYPE, HEARTBEAT);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAndHeadersAreMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreNotMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test");
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreMatchedButHeadersAreMissingInRecord() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      SourceRecord sourceRecord = createSourceRecord(null);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreMatchedButHeaderKeyIsNotMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT + "V1", null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithIncludeFilterTypeWhenTopicsAreMatchedButHeaderKeyValuePairIsNotMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, INCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      Headers headers = new ConnectHeaders().addString(MESSAGE_TYPE, HEARTBEAT + "V1");
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreNotSpecifiedAndHeadersAreMissingInRecord() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, HEARTBEAT);
      filter.configure(props);
      SourceRecord sourceRecord = createSourceRecord(null);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreNotSpecifiedAndHeaderKeyIsMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      filter.configure(props);
      Headers headers = new ConnectHeaders()
          .addString(MESSAGE_TYPE, HEARTBEAT + "V1")
          .add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreNotSpecifiedAndOnlyHeaderKeyIsMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreNotSpecifiedAndHeaderKeyValuePairIsMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      filter.configure(props);
      Headers headers = new ConnectHeaders().addString(MESSAGE_TYPE, HEARTBEAT);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAndHeadersAreMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), nullValue());
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreNotMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test");
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT, null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreMatchedButHeadersAreMissingInRecord() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      SourceRecord sourceRecord = createSourceRecord(null);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreMatchedButHeaderKeyIsNotMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      Headers headers = new ConnectHeaders().add(HEARTBEAT + "V1", null);
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testApplyWithExcludeFilterTypeWhenTopicsAreMatchedButHeaderKeyValuePairIsNotMatched() {
    try (HeaderFilter<SourceRecord> filter = new HeaderFilter<>()) {
      Map<String, String> props = new HashMap<>();
      props.put(TYPE_CONFIG, EXCLUDE);
      props.put(HEADERS_CONFIG, MESSAGE_TYPE + "=" + HEARTBEAT + "," + HEARTBEAT);
      props.put(TOPICS_CONFIG, "test.*");
      filter.configure(props);
      Headers headers = new ConnectHeaders().addString(MESSAGE_TYPE, HEARTBEAT + "V1");
      SourceRecord sourceRecord = createSourceRecord(headers);

      assertThat(filter.apply(sourceRecord), equalTo(sourceRecord));
    }
  }

  @Test
  public void testParseHeadersToFilterWhenOnlyHeaderKeyIsSpecified() {
    Map<String, Optional<String>> headersToFilter =
        parseHeadersToFilter(Collections.singletonList(HEARTBEAT));

    assertThat(headersToFilter, hasEntry(HEARTBEAT, Optional.empty()));
  }

  @Test
  public void testParseHeadersToFilterWhenHeaderKeyValuePairIsSpecified() {
    Map<String, Optional<String>> headersToFilter =
        parseHeadersToFilter(Collections.singletonList(MESSAGE_TYPE + "=" + HEARTBEAT));

    assertThat(headersToFilter, hasEntry(MESSAGE_TYPE, Optional.of(HEARTBEAT)));
  }

  @Test(expected = ConfigException.class)
  public void testParseHeadersToFilterWhenHeaderHasInvalidFormat() {
    parseHeadersToFilter(
        Collections.singletonList(String.join("=", MESSAGE_TYPE, HEARTBEAT, "v1")));
  }

  @Test
  public void testConvertForByteArrayValue() {
    String value = "heartbeat";
    Object converted = convert(value.getBytes(StandardCharsets.UTF_8));

    assertThat(converted, equalTo(value));
  }

  @Test
  public void testConvertForEmptyByteArrayValue() {
    Object converted = convert("".getBytes(StandardCharsets.UTF_8));

    assertThat(converted, nullValue());
  }

  @Test
  public void testConvertForByteBufferValue() {
    String value = "heartbeat";
    Object converted = convert(ByteBuffer.wrap(value.getBytes(StandardCharsets.UTF_8)));

    assertThat(converted, equalTo(value));
  }

  @Test
  public void testConvertForEmptyByteBufferValue() {
    Object converted = convert(ByteBuffer.wrap("".getBytes(StandardCharsets.UTF_8)));

    assertThat(converted, nullValue());
  }

  @Test
  public void testConvertForStringValue() {
    String value = "heartbeat";
    Object converted = convert(value);

    assertThat(converted, equalTo(value));
  }

  @Test
  public void testConvertForEmptyStringValue() {
    Object converted = convert("");

    assertThat(converted, nullValue());
  }

  @Test
  public void testConvertForBooleanValue() {
    boolean value = true;
    Object converted = convert(value);

    assertThat(converted, equalTo("true"));
  }

  @Test
  public void testConvertForByteValue() {
    byte value = 1;
    Object converted = convert(value);

    assertThat(converted, equalTo("1"));
  }

  @Test
  public void testConvertForShortValue() {
    short value = 1;
    Object converted = convert(value);

    assertThat(converted, equalTo("1"));
  }

  @Test
  public void testConvertForIntegerValue() {
    int value = 1;
    Object converted = convert(value);

    assertThat(converted, equalTo("1"));
  }

  @Test
  public void testConvertForLongValue() {
    long value = 1L;
    Object converted = convert(value);

    assertThat(converted, equalTo("1"));
  }

  @Test
  public void testConvertForFloatValue() {
    float value = 1.0f;
    Object converted = convert(value);

    assertThat(converted, equalTo("1.0"));
  }

  @Test
  public void testConvertForDoubleValue() {
    double value = 1.0d;
    Object converted = convert(value);

    assertThat(converted, equalTo("1.0"));
  }

  @Test
  public void testConvertForOtherValue() {
    Object value = new Object();
    Object converted = convert(value);

    assertThat(converted, equalTo(value));
  }
}
