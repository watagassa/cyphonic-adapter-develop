\c cyphonic cyphonic;

CREATE TABLE IF NOT EXISTS accounts (
  id BIGSERIAL NOT NULL,
  email VARCHAR(1024) NOT NULL,
  password VARCHAR(1024) NOT NULL,
  state SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id),
  UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS auth_tokens (
  id BIGSERIAL NOT NULL,
  account_id BIGINT CHECK (account_id >=0) REFERENCES accounts(id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  access_token VARCHAR(1600) NOT NULL,
  refresh_token VARCHAR(1600) NOT NULL,
  expire TIMESTAMP NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id),
  UNIQUE(account_id)
);

CREATE TABLE IF NOT EXISTS node_management_areas (
  id SERIAL NOT NULL,
  area VARCHAR(30) NOT NULL,
  fqdn VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS cache_informations (
  id SERIAL NOT NULL,
  zone VARCHAR(512) NOT NULL,
  node_id BYTEA NOT NULL,
  real_ipv4 BYTEA NOT NULL,
  real_ipv6 BYTEA NOT NULL,
  common_key BYTEA NOT NULL,
  common_key_cipher_type SMALLINT NOT NULL,
  common_key_length SMALLINT NOT NULL,
  common_key_expire TIMESTAMP NOT NULL,
  nms_flag BOOLEAN NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS device_types (
  id SERIAL NOT NULL,
  device_type VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS devices (
  id BIGSERIAL NOT NULL,
  version VARCHAR(10) NOT NULL,
  device_id VARCHAR(128) NOT NULL,
  account_id BIGINT CHECK (account_id>=0) REFERENCES accounts(id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  device_name VARCHAR(128) NOT NULL,
  fqdn VARCHAR(256) NOT NULL,
  node_management_area_id INTEGER CHECK (node_management_area_id>=0) REFERENCES node_management_areas (id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  device_type INTEGER CHECK (device_type>=0) REFERENCES device_types (id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  adapter_flag BOOLEAN NOT NULL,
  general_node_flag BOOLEAN NOT NULL,
  inter_node_certification_flag BOOLEAN NOT NULL,
  status SMALLINT CHECK (status>=0) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id),
  UNIQUE(device_id),
  UNIQUE(device_name),
  UNIQUE(fqdn)
);
CREATE INDEX ON devices(node_management_area_id);
CREATE INDEX ON devices(account_id);


CREATE TABLE IF NOT EXISTS device_passwords (
  id BIGSERIAL NOT NULL,
  device_id VARCHAR(128) REFERENCES devices(device_id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  password VARCHAR(1024) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id),
  UNIQUE(device_id)
);

CREATE TABLE IF NOT EXISTS fqdn_aliases (
  id BIGSERIAL NOT NULL,
  device_id VARCHAR(128) REFERENCES devices (device_id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  fqdn_alias VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);
CREATE INDEX ON fqdn_aliases (device_id);

CREATE TABLE IF NOT EXISTS general_devices (
  id BIGSERIAL NOT NULL,
  adapter_id VARCHAR(128) REFERENCES devices (device_id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  device_id VARCHAR(128) NOT NULL,
  mac_address VARCHAR(128) NOT NULL,
  enable_ipv6 BOOLEAN NOT NULL,
  auth_type SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);
CREATE INDEX ON general_devices (adapter_id);

CREATE TABLE IF NOT EXISTS digital_certificates (
  id BIGSERIAL NOT NULL,
  device_id VARCHAR(128) REFERENCES devices (device_id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  status SMALLINT CHECK (status>=0) NOT NULL,
  expire TIMESTAMP NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);
CREATE UNIQUE INDEX ON digital_certificates (device_id);

CREATE TABLE IF NOT EXISTS node_informations (
  id BIGSERIAL NOT NULL,
  fqdn VARCHAR(256) NOT NULL,
  node_id BYTEA NOT NULL,
  virtual_ipv4 BYTEA,
  virtual_ipv6 BYTEA,
  application_id VARCHAR(128) NOT NULL,
  application_port INTEGER CHECK (application_port>=0) NULL DEFAULT NULL,
  notification_type SMALLINT CHECK (notification_type>=0) NULL DEFAULT NULL,
  common_key BYTEA NOT NULL,
  common_key_cipher_type SMALLINT NOT NULL,
  common_key_length SMALLINT NOT NULL,
  common_key_expire TIMESTAMP NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);
CREATE UNIQUE INDEX ON node_informations (node_id);

CREATE TABLE IF NOT EXISTS apple_push_notification_services (
  id SERIAL NOT NULL,
  fqdn_mobile_node VARCHAR(256) NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  device_token VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS apple_push_notification_service_certification_authorities (
  id SERIAL NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  certification_authority_path VARCHAR(256) NOT NULL,
  passphrase VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS firebase_cloud_messagings (
  id SERIAL NOT NULL,
  fqdn_mobile_node VARCHAR(256) NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  registration_id VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS firebase_cloud_messaging_apis (
  id SERIAL NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  api_key VARCHAR(256) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS key_notifications (
  id SERIAL NOT NULL,
  version VARCHAR(10) NOT NULL,
  src_fqdn VARCHAR(256) NOT NULL,
  dest_fqdn VARCHAR(256) NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  service_id SMALLINT NOT NULL,
  service_item_id VARCHAR(512) NOT NULL,
  hash_id VARCHAR(256) NOT NULL,
  state SMALLINT NOT NULL,
  data_id INTEGER CHECK (data_id>=0) NOT NULL,
  data BYTEA NOT NULL,
  upload_time TIMESTAMP NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS node_addresses (
  id BIGSERIAL NOT NULL,
  node_id BYTEA REFERENCES node_informations (node_id) ON DELETE NO ACTION ON UPDATE NO ACTION NOT NULL,
  interface_id INTEGER CHECK (interface_id>=0) NOT NULL,
  real_ipv4 BYTEA NOT NULL,
  real_ipv6 BYTEA NOT NULL,
  nat_ipv4 BYTEA NOT NULL,
  nat_ipv6 BYTEA NOT NULL,
  nat_port INTEGER CHECK (nat_port>=0) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);
CREATE UNIQUE INDEX ON node_addresses (node_id);

CREATE TABLE IF NOT EXISTS notification_cancels (
  id BIGSERIAL NOT NULL,
  fqdn_mn VARCHAR(256) NOT NULL,
  application_id VARCHAR(128) NOT NULL,
  service_id SMALLINT NOT NULL,
  service_item_id VARCHAR(512) NOT NULL,
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS path_informations (
  id BIGSERIAL NOT NULL,
  path_id BYTEA NOT NULL,
  generating_flag BOOLEAN NOT NULL,
  mn_ipv4 BYTEA NOT NULL,
  mn_ipv6 BYTEA NOT NULL,
  mn_port INTEGER CHECK (mn_port>=0) NOT NULL,
  cn_ipv4 BYTEA NOT NULL,
  cn_ipv6 BYTEA NOT NULL,
  cn_port INTEGER CHECK (cn_port>=0) NOT NULL,
  tunnel_key BYTEA NOT NULL,
  tunnel_key_cipher_type SMALLINT NOT NULL,
  tunnel_key_length SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);