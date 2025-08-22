// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"fmt"
	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	username = "admin"
	password = "admin"
	caCrt    = "-----BEGIN CERTIFICATE-----\nMIIC8jCCAdqgAwIBAgIQQJXQsb78s3csoU7UFg9CAzANBgkqhkiG9w0BAQsFADAT\nMREwDwYDVQQDEwhrYWZrYS1jYTAeFw0yMjA0MjcxMzEzMDZaFw0yMzA0MjcxMzEz\nMDZaMBMxETAPBgNVBAMTCGthZmthLWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\nMIIBCgKCAQEAn949e5nhuNes3UmkY1h94dEwYX9/W9Hmt7Ye+A3ep/VftUlNH7WF\ni7jrmXvTAso/f8LI5P+PI37inIED17iBhcJpNxXx1/vU0fjy5hF0ZRN9UPMK5nzw\nTYJPQefXTU+w++vjlQ3XaysLOuyGZ1h8vCKzb6qJhnSu7JXbC4FWvPvvTLAmx8Mo\nOrobFqrqFvVxuN+1Qo6/7leo4/5yHoA9tWYPwE8/spGC5MzqYR4vGzhh9uXV36sa\nJiQjo9seJmLUnuOgbBDd4C4y/IjQ3GxRGurSGbwM8SYSAEZ8JX5oGDl2RFchnZCi\n8+ryXYTbE04kfalP/2Ew87XIfqHG6K6KzQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMC\nAqQwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMA8GA1UdEwEB/wQFMAMB\nAf8wDQYJKoZIhvcNAQELBQADggEBAHMrXZch3IrROLVwSGugmBid2iOBf/i9lX3H\nbd8ajr87Hd9xsUsusc2yzf0G9rk21NRJI40I191EnNMQJVv6D7p8BysUV75t4vC5\nKl77bFX24TlcHUL6LqJUAXvGsoEGbJ6lzJ72Rl4hiFmF1rvJgC9bSDbMop9fQF79\nFAp2wIxzEZl9PuWOBdZ1LKMKhFrW3LBw2PD4mjzZIsZgNVs2BcbHwNe0/jrYyRAQ\nojcHZKYEDpaZm36uXUgQDw6IZM7Dqrw3vycG5B0fh2G3n3APqIj4HAFDb1zuOM5T\npBOzGt+KLmIje04vrlaQsUntTCjYDF8SPQ/hOeV5XiPDaAc0z4c=\n-----END CERTIFICATE-----\n"
	tlsCert  = "-----BEGIN CERTIFICATE-----\nMIIE2DCCA8CgAwIBAgIRAJ4k1U6McZJE+k+bn5upiT0wDQYJKoZIhvcNAQELBQAw\nEzERMA8GA1UEAxMIa2Fma2EtY2EwHhcNMjIwNDI3MTMxMzA2WhcNMjMwNDI3MTMx\nMzA2WjAQMQ4wDAYDVQQDEwVrYWZrYTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC\nAQoCggEBAJ2cMLqmH/tNzrSQtv6oWsHsyTQPHYiCAdEc4VnxZr39kQlKyF2vT6Qz\nT2j1a5UZDdKo+8l5nmeXslitCGFaIZxeUviqu/OymI4B7iJ4fsneokfYK9Gsus7t\n/SKAb9RPvLnN/gqh/ruPCPJRtfkTU0TqpcXq9/ciiByhlVjsHSpN4rWbC/dIj3+W\nF1LAHh0lagzq8ptI5zycgLd5Wl9uhg7CxN93uD2fk75BsKnwtIfsSJH1Tq1yPyOL\nZSAaThUEhzxKi0A4KEaIPxoy/WR9BAAx+b7zVElyI0xjTP1H/U37aOECibE6rIvu\np2nruPyRKo261+DHZs3ffF/QqUq/27ECAwEAAaOCAigwggIkMA4GA1UdDwEB/wQE\nAwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIw\nADCCAeMGA1UdEQSCAdowggHWgglsb2NhbGhvc3SCBWthZmthghNrYWZrYS5rYWZr\nYS1zZXJ2aWNlghJrYWZrYS5rYWZrYS1icm9rZXKCIGthZmthLmthZmthLWJyb2tl\nci5rYWZrYS1zZXJ2aWNlghdrYWZrYS5rYWZrYS1zZXJ2aWNlLnN2Y4IHa2Fma2Et\nMYIVa2Fma2EtMS5rYWZrYS1zZXJ2aWNlgiJrYWZrYS0xLmthZmthLWJyb2tlci5r\nYWZrYS1zZXJ2aWNlggdrYWZrYS0yghVrYWZrYS0yLmthZmthLXNlcnZpY2WCImth\nZmthLTIua2Fma2EtYnJva2VyLmthZmthLXNlcnZpY2WCB2thZmthLTOCFWthZmth\nLTMua2Fma2Etc2VydmljZYIia2Fma2EtMy5rYWZrYS1icm9rZXIua2Fma2Etc2Vy\ndmljZYI3cGFhcy1taW5paGEta3ViZXJuZXRlcy5vcGVuc2hpZnQuc2RudGVzdC5u\nZXRjcmFja2VyLmNvbYJBZGFzaGJvYXJkLnBhYXMtbWluaWhhLWt1YmVybmV0ZXMu\nb3BlbnNoaWZ0LnNkbnRlc3QubmV0Y3JhY2tlci5jb22HBH8AAAGHBAppIHmHBApp\nIH6HBAppIH4wDQYJKoZIhvcNAQELBQADggEBABoy9+94IJ7pgYSlKfo2OyXkN0JH\nvn46+Vp48ZP7LA2OrYpjkVOpwICFl+ffGnCJ4O4dEeyas3LKHfG2OxOg0Uo+x5ob\ntwTltQI0ZVGX9RSnCJzIPqR2jgnD+5yWTbSvKkG2Yn0DAPcx+RezqMwUApujker2\nPwpoQQ3mwkyqkQgHt9SueTpPoOGiTUw6+L0t/DxXCELUXhSecfAXRgQFfPTQO1cU\nrCUgyDFv8L6ytIHbEPRPYTfQNnRXcKUpidDxuvvMsdr0iP/ZfR0S7NHccgKul1XG\nJr0c1jlpIY0gXnV0szhAHuhjs4VYd+nttVhGE7l8iGNb4wq3ot0GNo0r1C4=\n-----END CERTIFICATE-----\n"
	tlsKey   = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAnZwwuqYf+03OtJC2/qhawezJNA8diIIB0RzhWfFmvf2RCUrI\nXa9PpDNPaPVrlRkN0qj7yXmeZ5eyWK0IYVohnF5S+Kq787KYjgHuInh+yd6iR9gr\n0ay6zu39IoBv1E+8uc3+CqH+u48I8lG1+RNTROqlxer39yKIHKGVWOwdKk3itZsL\n90iPf5YXUsAeHSVqDOrym0jnPJyAt3laX26GDsLE33e4PZ+TvkGwqfC0h+xIkfVO\nrXI/I4tlIBpOFQSHPEqLQDgoRog/GjL9ZH0EADH5vvNUSXIjTGNM/Uf9Tfto4QKJ\nsTqsi+6naeu4/JEqjbrX4Mdmzd98X9CpSr/bsQIDAQABAoIBAADFAnfm18EiYCAB\nlOMpb0gDH/hhGoPQHrImsfL+esHyuwKQmunaMUlb9sdMa3oO5UJiENiq/1sCIpzM\nO34+MmvRChasvr4x4QzQdZk1fWj+7UvsgkpDzaW1A8dnWSRwPzupBdSHdfb0e9az\nD3Bn58AuZSDSROOwB3ocT95fSMUsGjjr4JIIMoO5/KHMbUPOL5weAglzAiYFzBaw\nFximnVfNntz5QyLjFXskZDRw3fNllyQx+mDJd8vUiRxpFl3Kcc7yl8/JXWCuFcX4\nJ/4nuFqUXCjGWPQYesLSPgV088lShBCojfTpgMY3mS4XSbpIR7o2vFU/LSTlhcRo\n0FL7LAECgYEAw2Xoou+7uhJg7biAjNx+auEM6PwZrWocnkK9tcxmIqqbmIGWsjwU\nfMFEG/81Y7kOEqWVsVlu2cpXbQo6qnamBVCERJQR6GBYUiKw/eNVQOQhusfTNKwn\nDbZHjihS5Zx8ViZUGOVcIUxnRnE5pe7Fkn5Cq1bmXmYjsoNoDfC0N8UCgYEAzn4E\narO8ZXzOecYzhERKqiD5HUGMWP9Xy6r9amGp/kHIPX3HVeL3mGhK5zN5Zv88LhhH\nJ+qJVE9WaodQnh24bZt0EZGbt90EXOxdOnSVwUUXp6hR8NwlmViBeKwtsqx0XP1i\nv5+AZRJuQtHr0cwwYt1/vI14uFjM24Pxnrc/pv0CgYEAtIeFRnUEFshAMaJTctGN\nIyZGjUPOXYA6bKXxLPRqMQE7vM2N86K6swDE8rD6HOau7994zGB6oFHoMGBRD4mL\nnkFj0xCS8wWA1HIk4I2XCNs9pppUsseTVYHh3p+251mLLvU+obnXQxSaHmUiBAL1\nG4H4CuHA+dqYhKgQDUEk4JkCgYAv8s8vv8C1iD+hw0ZfJkR4MOPnyTq/x7spTfE+\nbKM+qSPIM5a/+M4pk74g5bEBG69rvLN5L1roOuwEHJu5u4kB2qEfG0KfdTD4KuKT\nGlNT56lQgyNT3KrWatjVnpWV8bmrhiMSAAWecqMr3Pb3ZoSt0GVC8U7g763SI1dN\n1ZtwOQKBgQCYR5DrKlMxDDUACx3gCjeJer4ITCz4jTWE8bL6RR71yxEilqZNzpPI\nRmEglFqCWexsq0nbcBzqr+giuTOHFIEzSn00ASfbVvWmYGm36CH6fpLwVrU5v4uz\n/p71yMYs6E+IX0+MONejeSx7ugyPN65TSG2Nse9ASyH7ZSfVHlNkNg==\n-----END RSA PRIVATE KEY-----\n"
)

func TestNewKafkaClientConfigWhenSaslMechanismIsEmpty(t *testing.T) {
	saslSettings := &SaslSettings{
		Username: username,
		Password: password,
	}
	config, err := NewKafkaClientConfig(saslSettings, false, &SslCertificates{})
	assert.Nil(t, err)
	assert.Equal(t, true, config.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512), config.Net.SASL.Mechanism)
	assert.Equal(t, username, config.Net.SASL.User)
	assert.Equal(t, password, config.Net.SASL.Password)
	assert.NotNil(t, config.Net.SASL.SCRAMClientGeneratorFunc)
	assert.Equal(t, false, config.Net.TLS.Enable)
}

func TestNewKafkaClientConfigWhenSaslMechanismIsPlain(t *testing.T) {
	saslSettings := &SaslSettings{
		Mechanism: sarama.SASLTypePlaintext,
		Username:  username,
		Password:  password,
	}
	config, err := NewKafkaClientConfig(saslSettings, false, &SslCertificates{})
	assert.Nil(t, err)
	assert.Equal(t, true, config.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypePlaintext), config.Net.SASL.Mechanism)
	assert.Equal(t, username, config.Net.SASL.User)
	assert.Equal(t, password, config.Net.SASL.Password)
	assert.Nil(t, config.Net.SASL.SCRAMClientGeneratorFunc)
	assert.Equal(t, false, config.Net.TLS.Enable)
}

func TestNewKafkaClientConfigWhenSaslMechanismIsScramSha512(t *testing.T) {
	saslSettings := &SaslSettings{
		Mechanism: sarama.SASLTypeSCRAMSHA512,
		Username:  username,
		Password:  password,
	}
	config, err := NewKafkaClientConfig(saslSettings, false, &SslCertificates{})
	assert.Nil(t, err)
	assert.Equal(t, true, config.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512), config.Net.SASL.Mechanism)
	assert.Equal(t, username, config.Net.SASL.User)
	assert.Equal(t, password, config.Net.SASL.Password)
	assert.NotNil(t, config.Net.SASL.SCRAMClientGeneratorFunc)
	assert.Equal(t, false, config.Net.TLS.Enable)
}

func TestNewKafkaClientConfigWhenSaslMechanismIsScramSha256(t *testing.T) {
	saslSettings := &SaslSettings{
		Mechanism: sarama.SASLTypeSCRAMSHA256,
		Username:  username,
		Password:  password,
	}
	_, err := NewKafkaClientConfig(saslSettings, false, &SslCertificates{})
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Sprintf("cannot use given SASL Mechanism: %s", sarama.SASLTypeSCRAMSHA256), err.Error())
}

func TestNewKafkaClientConfigWhenSaslIsDisabledAndCaCertExists(t *testing.T) {
	sslCertificates := &SslCertificates{
		CaCert: []byte(caCrt),
	}
	config, err := NewKafkaClientConfig(&SaslSettings{}, true, sslCertificates)
	assert.Nil(t, err)
	assert.Equal(t, false, config.Net.SASL.Enable)
	assert.Equal(t, true, config.Net.TLS.Enable)
	assert.NotNil(t, config.Net.TLS.Config.RootCAs)
	assert.Len(t, config.Net.TLS.Config.Certificates, 0)
}

func TestNewKafkaClientConfigWhenSaslIsDisabledAndTlsEnabled(t *testing.T) {
	config, err := NewKafkaClientConfig(&SaslSettings{}, true, &SslCertificates{})
	assert.Nil(t, err)
	assert.Equal(t, false, config.Net.SASL.Enable)
	assert.Equal(t, true, config.Net.TLS.Enable)
}

func TestNewKafkaClientConfigWhenSaslIsDisabledAndCaCertAndTlsCertExist(t *testing.T) {
	sslCertificates := &SslCertificates{
		CaCert:  []byte(caCrt),
		TlsCert: []byte(tlsCert),
		TlsKey:  []byte(tlsKey),
	}
	config, err := NewKafkaClientConfig(&SaslSettings{}, true, sslCertificates)
	assert.Nil(t, err)
	assert.Equal(t, false, config.Net.SASL.Enable)
	assert.Equal(t, true, config.Net.TLS.Enable)
	assert.NotNil(t, config.Net.TLS.Config.RootCAs)
	assert.Len(t, config.Net.TLS.Config.Certificates, 1)
}

func TestNewKafkaClientConfigWhenSaslIsEnabledAndCaCertExists(t *testing.T) {
	saslSettings := &SaslSettings{
		Mechanism: sarama.SASLTypeSCRAMSHA512,
		Username:  username,
		Password:  password,
	}
	sslCertificates := &SslCertificates{
		CaCert: []byte(caCrt),
	}
	config, err := NewKafkaClientConfig(saslSettings, true, sslCertificates)
	assert.Nil(t, err)
	assert.Equal(t, true, config.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512), config.Net.SASL.Mechanism)
	assert.Equal(t, username, config.Net.SASL.User)
	assert.Equal(t, password, config.Net.SASL.Password)
	assert.NotNil(t, config.Net.SASL.SCRAMClientGeneratorFunc)
	assert.Equal(t, true, config.Net.TLS.Enable)
	assert.NotNil(t, config.Net.TLS.Config.RootCAs)
	assert.Len(t, config.Net.TLS.Config.Certificates, 0)
}

func TestNewKafkaClientConfigWhenSaslIsEnabledAndCaCertAndTlsCertExist(t *testing.T) {
	saslSettings := &SaslSettings{
		Mechanism: sarama.SASLTypeSCRAMSHA512,
		Username:  username,
		Password:  password,
	}
	sslCertificates := &SslCertificates{
		CaCert:  []byte(caCrt),
		TlsCert: []byte(tlsCert),
		TlsKey:  []byte(tlsKey),
	}
	config, err := NewKafkaClientConfig(saslSettings, true, sslCertificates)
	assert.Nil(t, err)
	assert.Equal(t, true, config.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512), config.Net.SASL.Mechanism)
	assert.Equal(t, username, config.Net.SASL.User)
	assert.Equal(t, password, config.Net.SASL.Password)
	assert.NotNil(t, config.Net.SASL.SCRAMClientGeneratorFunc)
	assert.Equal(t, true, config.Net.TLS.Enable)
	assert.NotNil(t, config.Net.TLS.Config.RootCAs)
	assert.Len(t, config.Net.TLS.Config.Certificates, 1)
}
