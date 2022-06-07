// Copyright 2020 Redpanda Data, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.md
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0

package config

import (
	"crypto/tls"
	"path"

	"github.com/spf13/afero"
	"github.com/twmb/tlscfg"
)

type Config struct {
	file       *Config
	loadedPath string

	NodeUUID             string          `yaml:"node_uuid,omitempty" json:"node_uuid"`
	Organization         string          `yaml:"organization,omitempty" json:"organization"`
	LicenseKey           string          `yaml:"license_key,omitempty" json:"license_key"`
	ClusterID            string          `yaml:"cluster_id,omitempty" json:"cluster_id"`
	ConfigFile           string          `yaml:"config_file" json:"config_file"`
	Redpanda             RedpandaConfig  `yaml:"redpanda" json:"redpanda"`
	Rpk                  RpkConfig       `yaml:"rpk" json:"rpk"`
	Pandaproxy           *Pandaproxy     `yaml:"pandaproxy,omitempty" json:"pandaproxy,omitempty"`
	PandaproxyClient     *KafkaClient    `yaml:"pandaproxy_client,omitempty" json:"pandaproxy_client,omitempty"`
	SchemaRegistry       *SchemaRegistry `yaml:"schema_registry,omitempty" json:"schema_registry,omitempty"`
	SchemaRegistryClient *KafkaClient    `yaml:"schema_registry_client,omitempty" json:"schema_registry_client,omitempty"`

	Other map[string]interface{} `yaml:",inline"`
}

// File returns the configuration as read from a file, with no defaults
// pre-deserializing and no overrides applied after. If the return is nil,
// no file was read.
func (c *Config) File() *Config {
	return c.file
}

type RedpandaConfig struct {
	Directory                  string                 `yaml:"data_directory" json:"data_directory"`
	ID                         int                    `yaml:"node_id" json:"node_id"`
	Rack                       string                 `yaml:"rack,omitempty" json:"rack"`
	SeedServers                []SeedServer           `yaml:"seed_servers" json:"seed_servers"`
	RPCServer                  SocketAddress          `yaml:"rpc_server" json:"rpc_server"`
	RPCServerTLS               []ServerTLS            `yaml:"rpc_server_tls,omitempty" json:"rpc_server_tls"`
	KafkaAPI                   []NamedSocketAddress   `yaml:"kafka_api" json:"kafka_api"`
	KafkaAPITLS                []ServerTLS            `yaml:"kafka_api_tls,omitempty" json:"kafka_api_tls"`
	AdminAPI                   []NamedSocketAddress   `yaml:"admin" json:"admin"`
	AdminAPITLS                []ServerTLS            `yaml:"admin_api_tls,omitempty" json:"admin_api_tls"`
	CoprocSupervisorServer     SocketAddress          `yaml:"coproc_supervisor_server,omitempty" json:"coproc_supervisor_server"`
	AdminAPIDocDir             string                 `yaml:"admin_api_doc_dir,omitempty" json:"admin_api_doc_dir"`
	DashboardDir               string                 `yaml:"dashboard_dir,omitempty" json:"dashboard_dir"`
	CloudStorageCacheDirectory string                 `yaml:"cloud_storage_cache_directory,omitempty" json:"cloud_storage_cache_directory"`
	AdvertisedRPCAPI           *SocketAddress         `yaml:"advertised_rpc_api,omitempty" json:"advertised_rpc_api,omitempty"`
	AdvertisedKafkaAPI         []NamedSocketAddress   `yaml:"advertised_kafka_api,omitempty" json:"advertised_kafka_api,omitempty"`
	DeveloperMode              bool                   `yaml:"developer_mode" json:"developer_mode"`
	Other                      map[string]interface{} `yaml:",inline"`
}

type Pandaproxy struct {
	PandaproxyAPI           []NamedSocketAddress   `yaml:"pandaproxy_api,omitempty" json:"pandaproxy_api,omitempty"`
	PandaproxyAPITLS        []ServerTLS            `yaml:"pandaproxy_api_tls,omitempty" json:"pandaproxy_api_tls,omitempty"`
	AdvertisedPandaproxyAPI []NamedSocketAddress   `yaml:"advertised_pandaproxy_api,omitempty" json:"advertised_pandaproxy_api,omitempty"`
	Other                   map[string]interface{} `yaml:",inline"`
}

type SchemaRegistry struct {
	SchemaRegistryAPI               []NamedSocketAddress `yaml:"schema_registry_api,omitempty" json:"schema_registry_api,omitempty"`
	SchemaRegistryAPITLS            []ServerTLS          `yaml:"schema_registry_api_tls,omitempty" json:"schema_registry_api_tls,omitempty"`
	SchemaRegistryReplicationFactor *int                 `yaml:"schema_registry_replication_factor,omitempty" json:"schema_registry_replication_factor,omitempty"`
}

type KafkaClient struct {
	Brokers       []SocketAddress        `yaml:"brokers,omitempty" json:"brokers,omitempty"`
	BrokerTLS     ServerTLS              `yaml:"broker_tls,omitempty" json:"broker_tls,omitempty"`
	SASLMechanism *string                `yaml:"sasl_mechanism,omitempty" json:"sasl_mechanism,omitempty"`
	SCRAMUsername *string                `yaml:"scram_username,omitempty" json:"scram_username,omitempty"`
	SCRAMPassword *string                `yaml:"scram_password,omitempty" json:"scram_password,omitempty"`
	Other         map[string]interface{} `yaml:",inline"`
}

type SeedServer struct {
	Host SocketAddress `yaml:"host" json:"host"`
}

type SocketAddress struct {
	Address string `yaml:"address" json:"address"`
	Port    int    `yaml:"port" json:"port"`
}

type NamedSocketAddress struct {
	Address string `yaml:"address" json:"address"`
	Port    int    `yaml:"port" json:"port"`
	Name    string `yaml:"name,omitempty" json:"name,omitempty"`
}

type TLS struct {
	KeyFile        string `yaml:"key_file,omitempty" json:"key_file"`
	CertFile       string `yaml:"cert_file,omitempty" json:"cert_file"`
	TruststoreFile string `yaml:"truststore_file,omitempty" json:"truststore_file"`
}

func (t *TLS) Config(fs afero.Fs) (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}
	return tlscfg.New(
		tlscfg.WithFS(
			tlscfg.FuncFS(func(path string) ([]byte, error) {
				return afero.ReadFile(fs, path)
			}),
		),
		tlscfg.MaybeWithDiskCA(
			t.TruststoreFile,
			tlscfg.ForClient,
		),
		tlscfg.MaybeWithDiskKeyPair(
			t.CertFile,
			t.KeyFile,
		),
	)
}

type ServerTLS struct {
	Name              string                 `yaml:"name,omitempty" json:"name"`
	KeyFile           string                 `yaml:"key_file,omitempty" json:"key_file"`
	CertFile          string                 `yaml:"cert_file,omitempty" json:"cert_file"`
	TruststoreFile    string                 `yaml:"truststore_file,omitempty" json:"truststore_file"`
	Enabled           bool                   `yaml:"enabled,omitempty" json:"enabled"`
	RequireClientAuth bool                   `yaml:"require_client_auth,omitempty" json:"require_client_auth"`
	Other             map[string]interface{} `yaml:",inline" `
}

type RpkConfig struct {
	// Deprecated 2021-07-1
	TLS *TLS `yaml:"tls,omitempty" json:"tls"`
	// Deprecated 2021-07-1
	SASL *SASL `yaml:"sasl,omitempty" json:"sasl,omitempty"`

	KafkaAPI                 RpkKafkaAPI `yaml:"kafka_api,omitempty" json:"kafka_api"`
	AdminAPI                 RpkAdminAPI `yaml:"admin_api,omitempty" json:"admin_api"`
	AdditionalStartFlags     []string    `yaml:"additional_start_flags,omitempty"  json:"additional_start_flags"`
	EnableUsageStats         bool        `yaml:"enable_usage_stats" json:"enable_usage_stats"`
	TuneNetwork              bool        `yaml:"tune_network" json:"tune_network"`
	TuneDiskScheduler        bool        `yaml:"tune_disk_scheduler" json:"tune_disk_scheduler"`
	TuneNomerges             bool        `yaml:"tune_disk_nomerges" json:"tune_disk_nomerges"`
	TuneDiskWriteCache       bool        `yaml:"tune_disk_write_cache" json:"tune_disk_write_cache"`
	TuneDiskIrq              bool        `yaml:"tune_disk_irq" json:"tune_disk_irq"`
	TuneFstrim               bool        `yaml:"tune_fstrim" json:"tune_fstrim"`
	TuneCPU                  bool        `yaml:"tune_cpu" json:"tune_cpu"`
	TuneAioEvents            bool        `yaml:"tune_aio_events" json:"tune_aio_events"`
	TuneClocksource          bool        `yaml:"tune_clocksource" json:"tune_clocksource"`
	TuneSwappiness           bool        `yaml:"tune_swappiness" json:"tune_swappiness"`
	TuneTransparentHugePages bool        `yaml:"tune_transparent_hugepages" json:"tune_transparent_hugepages"`
	EnableMemoryLocking      bool        `yaml:"enable_memory_locking" json:"enable_memory_locking"`
	TuneCoredump             bool        `yaml:"tune_coredump" json:"tune_coredump"`
	CoredumpDir              string      `yaml:"coredump_dir,omitempty" json:"coredump_dir"`
	TuneBallastFile          bool        `yaml:"tune_ballast_file" json:"tune_ballast_file"`
	BallastFilePath          string      `yaml:"ballast_file_path,omitempty" json:"ballast_file_path"`
	BallastFileSize          string      `yaml:"ballast_file_size,omitempty" json:"ballast_file_size"`
	WellKnownIo              string      `yaml:"well_known_io,omitempty" json:"well_known_io"`
	Overprovisioned          bool        `yaml:"overprovisioned" json:"overprovisioned"`
	SMP                      *int        `yaml:"smp,omitempty" json:"smp,omitempty"`
}

type RpkKafkaAPI struct {
	Brokers []string `yaml:"brokers,omitempty" json:"brokers"`
	TLS     *TLS     `yaml:"tls,omitempty" json:"tls"`
	SASL    *SASL    `yaml:"sasl,omitempty" json:"sasl,omitempty"`
}

type RpkAdminAPI struct {
	Addresses []string `yaml:"addresses,omitempty" json:"addresses"`
	TLS       *TLS     `yaml:"tls,omitempty" json:"tls"`
}

type SASL struct {
	User      string `yaml:"user,omitempty" json:"user,omitempty"`
	Password  string `yaml:"password,omitempty" json:"password,omitempty"`
	Mechanism string `yaml:"type,omitempty" json:"type,omitempty"`
}

func (c *Config) PIDFile() string {
	return path.Join(c.Redpanda.Directory, "pid.lock")
}
