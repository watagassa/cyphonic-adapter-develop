\c cyphonic cyphonic;

INSERT INTO
  device_types (id, device_type)
VALUES
  (1, 'linux'),
  (2, 'windows'),
  (3, 'darwin'),
  (4, 'ios'),
  (5, 'android');

INSERT INTO
  node_management_areas(id,area, fqdn)
VALUES
  (
    1,
    'asia',
    'nms01.local.cyphonic.org'
  );

INSERT INTO
  cache_informations(
    id,
    zone,
    node_id,
    real_ipv4,
    real_ipv6,
    common_key,
    common_key_cipher_type,
    common_key_length,
    common_key_expire,
    nms_flag
  )
VALUES
  (
    1,
    'nms01.local.cyphonic.org',
    decode('1CABB67FE418524DA02AA3B3D7BC51CA', 'hex'),
    decode('0A0064CA', 'hex'),
    decode('FDE40DB8000000000000000000000202', 'hex'),
    decode('188A6638FA374DC486F5BF7A17A9AA2F', 'hex'),
    0,
    16,
    clock_timestamp() + cast('5 months' as INTERVAL),
    true
  ),
  (
    2,
    'trs01.local.cyphonic.org',
    decode('0B5F32C2E9E85A0593E41834518EAEC7', 'hex'),
    decode('0A0064CB', 'hex'),
    decode('FDE40DB8000000000000000000000203', 'hex'),
    decode('DE60FBB82A214297BD6C38B1142C99B3', 'hex'),
    0,
    16,
    clock_timestamp() + cast('5 months' as INTERVAL),
    false
  );